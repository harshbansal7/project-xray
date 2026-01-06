"""
HTTP client and async buffer for sending data to X-Ray API.
"""

import atexit
import json
import logging
import queue
import threading
import time
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING

import httpx

from xray_sdk.config import get_config, XRayConfig
from xray_sdk.models import TraceData, EventData

if TYPE_CHECKING:
    from xray_sdk.decision import Decision


logger = logging.getLogger("xray_sdk")


class XRayClient:
    """
    HTTP client for communicating with X-Ray API.
    
    Handles batching, async sending, retries, and fallback to local storage.
    """
    
    def __init__(self, config: Optional[XRayConfig] = None):
        """Initialize the client with configuration."""
        self._config = config or get_config()
        self._queue: queue.Queue[Tuple[str, Any]] = queue.Queue()
        self._shutdown = threading.Event()
        self._flush_thread: Optional[threading.Thread] = None
        self._http_client: Optional[httpx.Client] = None
        
        if self._config.async_send:
            self._start_flush_thread()
    
    def _start_flush_thread(self) -> None:
        """Start the background flush thread."""
        self._flush_thread = threading.Thread(
            target=self._flush_loop,
            daemon=True,
            name="xray-flush"
        )
        self._flush_thread.start()
        
        # Register cleanup on exit
        atexit.register(self.shutdown)
    
    def _get_http_client(self) -> httpx.Client:
        """Get or create HTTP client."""
        if self._http_client is None:
            headers = {}
            if self._config.api_key:
                headers["Authorization"] = f"Bearer {self._config.api_key}"
            headers["Content-Type"] = "application/json"
            
            self._http_client = httpx.Client(
                base_url=self._config.endpoint,
                timeout=self._config.timeout_ms / 1000,
                headers=headers,
            )
        return self._http_client
    
    def queue_trace_start(self, trace: TraceData) -> None:
        """Queue a trace start event."""
        self._queue.put(("trace_start", trace.model_dump(mode="json")))
        self._maybe_sync_flush()
    
    def queue_trace_end(self, trace: TraceData) -> None:
        """Queue a trace end event."""
        self._queue.put(("trace_end", trace.model_dump(mode="json")))
        self._maybe_sync_flush()
    
    def queue_event(self, event: EventData) -> None:
        """Queue an event."""
        self._queue.put(("event", event.model_dump(mode="json")))
        self._maybe_sync_flush()
    
    def queue_decisions(self, event_id: str, decisions: List["Decision"]) -> None:
        """Queue decisions for an event."""
        self._queue.put(("decisions", {
            "event_id": event_id,
            "decisions": [d.to_dict() for d in decisions]
        }))
        self._maybe_sync_flush()
    
    def _maybe_sync_flush(self) -> None:
        """Flush synchronously if not in async mode."""
        if not self._config.async_send:
            self._flush_now()
    
    def _flush_loop(self) -> None:
        """Background thread that batches and sends data."""
        batch: List[Tuple[str, Any]] = []
        last_flush = time.time()
        
        while not self._shutdown.is_set():
            try:
                # Non-blocking get with timeout
                item = self._queue.get(timeout=0.1)
                batch.append(item)
            except queue.Empty:
                pass
            
            # Check if we should flush
            should_flush = (
                len(batch) >= self._config.batch_size or
                (batch and time.time() - last_flush >= self._config.flush_interval_seconds)
            )
            
            if should_flush and batch:
                self._send_batch(batch)
                batch = []
                last_flush = time.time()
        
        # Flush remaining on shutdown
        if batch:
            self._send_batch(batch)
    
    def _flush_now(self) -> None:
        """Flush all queued items immediately."""
        batch: List[Tuple[str, Any]] = []
        
        while True:
            try:
                item = self._queue.get_nowait()
                batch.append(item)
            except queue.Empty:
                break
        
        if batch:
            self._send_batch(batch)
    
    def _send_batch(self, batch: List[Tuple[str, Any]]) -> None:
        """Send a batch of items to the API."""
        if not batch:
            return
        
        # Group by type, deduplicating traces by trace_id (keep latest)
        trace_map: Dict[str, Dict] = {}  # trace_id -> trace data
        event_map: Dict[str, Dict] = {}  # event_id -> event data
        all_decisions: List[Dict] = []
        
        for item_type, data in batch:
            if item_type in ("trace_start", "trace_end"):
                # Deduplicate: later entries overwrite earlier ones
                trace_id = data.get("trace_id")
                if trace_id:
                    trace_map[trace_id] = data
            elif item_type == "event":
                # Deduplicate events too
                event_id = data.get("event_id")
                if event_id:
                    event_map[event_id] = data
            elif item_type == "decisions":
                all_decisions.extend(data["decisions"])
        
        traces = list(trace_map.values())
        events = list(event_map.values())
        
        try:
            client = self._get_http_client()
            
            # Send traces
            if traces:
                self._send_with_retry(client, "/traces/batch", {"traces": traces})
            
            # Send events
            if events:
                self._send_with_retry(client, "/events/batch", {"events": events})
            
            # Send decisions
            if all_decisions:
                self._send_with_retry(client, "/decisions/batch", {"decisions": all_decisions})
            
            if self._config.debug:
                logger.debug(f"Sent batch: {len(traces)} traces, {len(events)} events, {len(all_decisions)} decisions")
        
        except Exception as e:
            logger.warning(f"Failed to send batch: {e}")
            self._handle_failure(batch)
    
    def _send_with_retry(self, client: httpx.Client, path: str, data: Dict, retries: int = 3) -> None:
        """Send data with retries."""
        last_error = None
        
        for attempt in range(retries):
            try:
                response = client.post(path, json=data)
                response.raise_for_status()
                return
            except httpx.HTTPStatusError as e:
                last_error = e
                if self._config.debug:
                    logger.error(f"HTTP {e.response.status_code}: {e.response.text}")
                if e.response.status_code < 500:
                    # Client error, don't retry
                    raise
            except httpx.RequestError as e:
                last_error = e
            
            # Wait before retry (exponential backoff)
            if attempt < retries - 1:
                time.sleep(0.1 * (2 ** attempt))
        
        raise last_error  # type: ignore
    
    def _handle_failure(self, batch: List[Tuple[str, Any]]) -> None:
        """Handle failed batch based on fallback config."""
        if self._config.fallback == "none":
            return
        
        if self._config.fallback == "local_file":
            self._write_to_file(batch)
        elif self._config.fallback == "memory":
            # Re-queue for later retry (with limit to prevent memory bloat)
            if self._queue.qsize() < 10000:
                for item in batch:
                    self._queue.put(item)
    
    def _write_to_file(self, batch: List[Tuple[str, Any]]) -> None:
        """Write failed batch to local file."""
        if not self._config.fallback_path:
            return
        
        path = Path(self._config.fallback_path)
        path.mkdir(parents=True, exist_ok=True)
        
        filename = path / f"xray_{int(time.time() * 1000)}.json"
        
        try:
            with open(filename, "w") as f:
                json.dump([{"type": t, "data": d} for t, d in batch], f)
            
            if self._config.debug:
                logger.debug(f"Wrote fallback to {filename}")
        except Exception as e:
            logger.error(f"Failed to write fallback: {e}")
    
    def shutdown(self) -> None:
        """Shutdown the client gracefully."""
        self._shutdown.set()
        
        if self._flush_thread and self._flush_thread.is_alive():
            self._flush_thread.join(timeout=5.0)
        
        if self._http_client:
            self._http_client.close()
            self._http_client = None


# Global client instance
_client: Optional[XRayClient] = None


def get_client() -> XRayClient:
    """Get or create the global client instance."""
    global _client
    if _client is None:
        _client = XRayClient()
    return _client


def reset_client() -> None:
    """Reset the global client. Mainly for testing."""
    global _client
    if _client:
        _client.shutdown()
    _client = None
