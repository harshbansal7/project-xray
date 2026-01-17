// X-Ray API Client
// Handles all communication with the X-Ray backend

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export interface Trace {
  trace_id: string;
  pipeline_id: string;
  started_at: string;
  ended_at?: string;
  status: string;
  metadata?: Record<string, unknown>;
  input_data?: Record<string, unknown>;
  tags?: string[];
}

export interface Event {
  event_id: string;
  trace_id: string;
  parent_event_id?: string;
  step_type: string;
  capture_mode: string;
  input_count?: number;
  output_count?: number;
  input_sample?: unknown[];
  output_sample?: unknown[];
  metrics?: Record<string, unknown>;
  annotations?: Record<string, unknown>;
  started_at: string;
  ended_at?: string;
}

export interface Decision {
  decision_id: string;
  event_id: string;
  trace_id: string;
  item_id: string;
  outcome: string;
  reason_code?: string;
  reason_detail?: string;
  scores?: Record<string, number>;
  item_snapshot?: Record<string, unknown>;
  timestamp: string;
}

export interface TraceWithEvents {
  trace: Trace;
  events: Event[];
  decisions?: Record<string, Decision[]>;
}

export interface TracePage {
  results: Trace[];
  count: number;
  next_cursor?: string;
}

export interface DecisionPage {
  results?: Decision[];
  decisions?: Decision[];
  next_cursor?: string;
  count?: number;
}

// API Functions

export async function getHealth(): Promise<{ status: string }> {
  const res = await fetch(`${API_BASE.replace('/api/v1', '')}/health`);
  return res.json();
}

export async function getTraces(params?: {
  pipeline_id?: string;
  status?: string;
  tags?: string[];
  metadata?: Record<string, string>;
  limit?: number;
}): Promise<TracePage> {
  const searchParams = new URLSearchParams();
  if (params?.pipeline_id) searchParams.set('pipeline_id', params.pipeline_id);
  if (params?.status) searchParams.set('status', params.status);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.tags && params.tags.length > 0) searchParams.set('tags', params.tags.join(','));
  
  if (params?.metadata) {
    Object.entries(params.metadata).forEach(([key, value]) => {
      searchParams.set(`meta:${key}`, value);
    });
  }
  
  const url = `${API_BASE}/traces${searchParams.toString() ? '?' + searchParams.toString() : ''}`;
  const res = await fetch(url);
  return res.json();
}

export async function getTrace(traceId: string): Promise<TraceWithEvents> {
  const res = await fetch(`${API_BASE}/traces/${traceId}`);
  return res.json();
}

export async function getEventsByTrace(traceId: string): Promise<Event[]> {
  const res = await fetch(`${API_BASE}/traces/${traceId}/events`);
  const data = await res.json();
  return data.events || [];
}

export async function getEvents(params?: {
  pipeline_id?: string;
  step_type?: string;
  limit?: number;
}): Promise<{ events: Event[] }> {
  const searchParams = new URLSearchParams();
  if (params?.pipeline_id) searchParams.set('pipeline_id', params.pipeline_id);
  if (params?.step_type) searchParams.set('step_type', params.step_type);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  
  const res = await fetch(`${API_BASE}/query/events?${searchParams.toString()}`);
  const data = await res.json();
  // Map 'results' from PaginatedResponse to 'events' expected by frontend
  return { events: data.results || data.events || [] };
}

export async function getDecisionsByEvent(
  traceId: string,
  eventId: string,
  params?: { outcome?: string }
): Promise<DecisionPage> {
  const searchParams = new URLSearchParams();
  if (params?.outcome) searchParams.set('outcome', params.outcome);
  
  const url = `${API_BASE}/traces/${traceId}/events/${eventId}/decisions${
    searchParams.toString() ? '?' + searchParams.toString() : ''
  }`;
  const res = await fetch(url);
  const data = await res.json();
  return { ...data, decisions: data.results || data.decisions || [] };
}

export async function queryEvents(params: {
  step_type?: string;
  pipeline_id?: string;
  min_reduction_ratio?: number;
  limit?: number;
}): Promise<{ events: Event[] }> {
  const searchParams = new URLSearchParams();
  if (params.step_type) searchParams.set('step_type', params.step_type);
  if (params.pipeline_id) searchParams.set('pipeline_id', params.pipeline_id);
  if (params.min_reduction_ratio) searchParams.set('min_reduction_ratio', params.min_reduction_ratio.toString());
  if (params.limit) searchParams.set('limit', params.limit.toString());
  
  const res = await fetch(`${API_BASE}/query/events?${searchParams.toString()}`);
  return res.json();
}

export async function queryDecisions(params: {
  pipeline_id?: string;
  step_type?: string;
  outcome?: string;
  limit?: number;
}): Promise<DecisionPage> {
  const searchParams = new URLSearchParams();
  if (params.pipeline_id) searchParams.set('pipeline_id', params.pipeline_id);
  if (params.step_type) searchParams.set('step_type', params.step_type);
  if (params.outcome) searchParams.set('outcome', params.outcome);
  if (params.limit) searchParams.set('limit', params.limit.toString());
  
  const res = await fetch(`${API_BASE}/query/decisions?${searchParams.toString()}`);
  const data = await res.json();
  // Map 'results' from PaginatedResponse to 'decisions' expected by frontend
  return { ...data, decisions: data.results || data.decisions || [] };
}

// Utility functions

export function formatDuration(startedAt: string, endedAt?: string): string {
  if (!endedAt) return 'running...';
  const start = new Date(startedAt).getTime();
  const end = new Date(endedAt).getTime();
  const duration = end - start;
  
  if (duration < 1000) return `${duration}ms`;
  if (duration < 60000) return `${(duration / 1000).toFixed(1)}s`;
  return `${(duration / 60000).toFixed(1)}m`;
}

export function getStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'success':
      return 'text-green-500';
    case 'failed':
    case 'error':
      return 'text-red-500';
    case 'running':
      return 'text-blue-500';
    default:
      return 'text-gray-500';
  }
}

export function getStatusBgColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'success':
      return 'bg-green-500/10 text-green-500 border-green-500/20';
    case 'failed':
    case 'error':
      return 'bg-red-500/10 text-red-500 border-red-500/20';
    case 'running':
      return 'bg-blue-500/10 text-blue-500 border-blue-500/20';
    default:
      return 'bg-gray-500/10 text-gray-500 border-gray-500/20';
  }
}
