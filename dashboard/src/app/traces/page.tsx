'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Search, Loader2, CheckCircle2, XCircle, Filter } from 'lucide-react';
import { getTraces, Trace, formatDuration } from '@/utils/api';
import { format } from 'date-fns';

export default function TracesPage() {
  const searchParams = useSearchParams();
  const pipelineFilter = searchParams.get('pipeline_id') || '';
  
  const [traces, setTraces] = useState<Trace[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedPipeline, setSelectedPipeline] = useState(pipelineFilter);
  const [pipelines, setPipelines] = useState<string[]>([]);

  useEffect(() => {
    async function load() {
      try {
        setLoading(true);
        const params: { pipeline_id?: string; status?: string; limit?: number } = { limit: 100 };
        if (selectedPipeline) params.pipeline_id = selectedPipeline;
        if (statusFilter) params.status = statusFilter;
        
        const response = await getTraces(params);
        const traceList = response.results || [];
        setTraces(traceList);
        setPipelines(Array.from(new Set(traceList.map(t => t.pipeline_id))));
      } catch (err) {
        console.error('Failed to load:', err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [selectedPipeline, statusFilter]);

  const filtered = traces.filter(t => {
    if (!search) return true;
    const s = search.toLowerCase();
    return t.trace_id.toLowerCase().includes(s) || t.pipeline_id.toLowerCase().includes(s);
  });

  return (
    <div className="w-full max-w-none px-4 sm:px-6 lg:px-8 2xl:px-10 py-4 sm:py-6 animate-fade-in">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-[var(--text-primary)]">Traces</h1>
        <p className="text-sm text-[var(--text-tertiary)] mt-1">Pipeline execution history</p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-4">
        <div className="relative flex-1 min-w-[220px] sm:min-w-[280px] lg:min-w-[360px] 2xl:min-w-[460px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
          <input
            type="text"
            placeholder="Search traces..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full h-9 pl-10 pr-3 text-sm bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg placeholder:text-[var(--text-tertiary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/30 focus:border-[var(--accent)]"
          />
        </div>
        <select
          value={selectedPipeline}
          onChange={e => setSelectedPipeline(e.target.value)}
          className="h-9 text-sm min-w-[160px] max-w-[320px]"
        >
          <option value="">All Pipelines</option>
          {pipelines.map(p => <option key={p} value={p}>{p}</option>)}
        </select>
        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value)}
          className="h-9 text-sm min-w-[140px]"
        >
          <option value="">All Status</option>
          <option value="running">Running</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
        </select>
      </div>

      <p className="text-xs text-[var(--text-tertiary)] mb-3">
        {filtered.length} traces
      </p>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="w-5 h-5 animate-spin text-[var(--text-tertiary)]" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="card p-12 text-center">
          <Filter className="w-8 h-8 mx-auto mb-2 text-[var(--text-tertiary)]" />
          <p className="text-[var(--text-secondary)]">No traces found</p>
        </div>
      ) : (
        <div className="card overflow-x-auto">
          <table className="w-full min-w-[900px]">
            <thead>
              <tr>
                <th>Status</th>
                <th>Pipeline</th>
                <th>Trace ID</th>
                <th>Started</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(trace => (
                <tr key={trace.trace_id}>
                  <td><StatusIcon status={trace.status} /></td>
                  <td className="text-[var(--text-primary)]">{trace.pipeline_id}</td>
                  <td>
                    <Link 
                      href={`/traces/${trace.trace_id}`}
                      className="font-mono text-[var(--accent)] hover:underline"
                    >
                      {trace.trace_id.slice(0, 12)}...
                    </Link>
                  </td>
                  <td>{format(new Date(trace.started_at), 'MMM d, HH:mm:ss')}</td>
                  <td>{formatDuration(trace.started_at, trace.ended_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
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
