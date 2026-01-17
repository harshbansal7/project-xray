'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { 
  ArrowLeft, 
  Clock,
  CheckCircle2,
  XCircle,
  Loader2,
  ChevronDown,
  ChevronRight,
  ArrowUpRight
} from 'lucide-react';
import { 
  getTrace, 
  getDecisionsByEvent,
  Trace, 
  Event, 
  Decision,
  formatDuration 
} from '@/utils/api';
import { getOutcomeColor } from '@/utils/colors';
import { format } from 'date-fns';
import DecisionStream from '@/components/trace/DecisionStream';

interface EventWithDecisions extends Event {
  decisions?: Decision[];
  loadingDecisions?: boolean;
}

export default function TraceDetailPage() {
  const params = useParams();
  const traceId = params.id as string;
  
  const [trace, setTrace] = useState<Trace | null>(null);
  const [events, setEvents] = useState<EventWithDecisions[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  useEffect(() => {
    async function load() {
      try {
        const data = await getTrace(traceId);
        setTrace(data.trace);
        
        // Pre-populate events with decisions from the response
        const eventsWithDecisions = (data.events || []).map(event => ({
          ...event,
          decisions: data.decisions?.[event.event_id] || []
        }));
        setEvents(eventsWithDecisions);
      } catch (err) {
        console.error('Failed to load:', err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [traceId]);

  const toggleEvent = async (eventId: string) => {
    const newExpanded = new Set(expanded);
    
    if (newExpanded.has(eventId)) {
      newExpanded.delete(eventId);
    } else {
      newExpanded.add(eventId);
      
      const event = events.find(e => e.event_id === eventId);
      if (event && !event.decisions) {
        setEvents(prev => prev.map(e => 
          e.event_id === eventId ? { ...e, loadingDecisions: true } : e
        ));
        
        try {
          const data = await getDecisionsByEvent(traceId, eventId);
          setEvents(prev => prev.map(e => 
            e.event_id === eventId 
              ? { ...e, decisions: data.decisions || [], loadingDecisions: false } 
              : e
          ));
        } catch {
          setEvents(prev => prev.map(e => 
            e.event_id === eventId ? { ...e, loadingDecisions: false } : e
          ));
        }
      }
    }
    
    setExpanded(newExpanded);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-6 h-6 animate-spin text-[var(--text-tertiary)]" />
      </div>
    );
  }

  if (!trace) {
    return (
      <div className="p-6">
        <Link href="/traces" className="inline-flex items-center gap-1.5 text-sm text-[var(--text-tertiary)] hover:text-[var(--text-primary)] mb-6">
          <ArrowLeft className="w-4 h-4" /> Traces
        </Link>
        <div className="card p-12 text-center">
          <p className="text-[var(--text-secondary)]">Trace not found</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl animate-fade-in">
      {/* Header */}
      <Link href="/traces" className="inline-flex items-center gap-1.5 text-sm text-[var(--text-tertiary)] hover:text-[var(--text-primary)] mb-3">
        <ArrowLeft className="w-4 h-4" /> Traces
      </Link>

      <div className="card p-4 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <StatusIcon status={trace.status} />
              <span className="text-sm font-medium text-[var(--text-primary)]">{trace.pipeline_id}</span>
            </div>
            <p className="text-xs text-[var(--text-tertiary)] font-mono">{trace.trace_id}</p>
          </div>
          <div className="text-right text-xs">
            <p className="text-[var(--text-tertiary)]">
              {format(new Date(trace.started_at), 'MMM d, yyyy HH:mm:ss')}
            </p>
            <p className="text-[var(--text-secondary)] font-medium mt-0.5">
              {formatDuration(trace.started_at, trace.ended_at)}
            </p>
          </div>
        </div>
      </div>

      {/* Events Timeline */}
      <h2 className="text-sm font-medium text-[var(--text-primary)] mb-3">Events ({events.length})</h2>
      
      {events.length === 0 ? (
        <div className="card p-8 text-center text-[var(--text-tertiary)]">
          No events recorded
        </div>
      ) : (
        <div className="space-y-2">
          {events.map(event => {
            const isExpanded = expanded.has(event.event_id);
            const reduction = event.input_count && event.output_count && event.input_count > 0
              ? 1 - (event.output_count / event.input_count)
              : null;
            
            return (
              <div key={event.event_id} className="card">
                <button
                  onClick={() => toggleEvent(event.event_id)}
                  className="w-full flex items-center justify-between p-3 hover:bg-[var(--bg-tertiary)] transition-colors"
                >
                  <div className="flex items-center gap-3">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4 text-[var(--text-tertiary)]" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-[var(--text-tertiary)]" />
                    )}
                    <span className="text-sm text-[var(--text-primary)]">{event.step_type}</span>
                  </div>
                  <div className="flex items-center gap-4 text-xs">
                    {event.input_count !== undefined && event.output_count !== undefined && (
                      <span className="text-[var(--text-secondary)]">
                        {event.input_count} → {event.output_count}
                        {reduction !== null && reduction > 0 && (
                          <span className={`ml-1.5 ${reduction >= 0.8 ? 'text-[var(--error)]' : reduction >= 0.5 ? 'text-[var(--warning)]' : 'text-[var(--success)]'}`}>
                            ({(reduction * 100).toFixed(0)}%)
                          </span>
                        )}
                      </span>
                    )}
                    <span className="text-[var(--text-tertiary)]">
                      {formatDuration(event.started_at, event.ended_at)}
                    </span>
                  </div>
                </button>

                {isExpanded && (
                  <div className="px-4 pb-4 pt-2 border-t border-[var(--border-primary)]">
                    {event.annotations && Object.keys(event.annotations).length > 0 && (
                      <div className="mb-3">
                        <p className="text-xs text-[var(--text-tertiary)] mb-1">Annotations</p>
                        <pre className="text-xs text-[var(--text-secondary)] bg-[var(--bg-tertiary)] p-2 rounded overflow-x-auto">
                          {JSON.stringify(event.annotations, null, 2)}
                        </pre>
                      </div>
                    )}

                    {event.loadingDecisions ? (
                      <div className="flex items-center gap-2 text-xs text-[var(--text-tertiary)]">
                        <Loader2 className="w-3 h-3 animate-spin" /> Loading decisions...
                      </div>
                    ) : event.decisions && event.decisions.length > 0 ? (
                      <DecisionStream decisions={event.decisions} />
                    ) : (
                      <p className="text-xs text-[var(--text-tertiary)]">No decisions recorded</p>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function StatusIcon({ status }: { status: string }) {
  if (status === 'completed') return <CheckCircle2 className="w-4 h-4 text-[var(--success)]" />;
  if (status === 'failed') return <XCircle className="w-4 h-4 text-[var(--error)]" />;
  return <Loader2 className="w-4 h-4 text-[var(--accent)] animate-spin" />;
}
