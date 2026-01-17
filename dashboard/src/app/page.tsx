'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { 
  Activity, 
  ArrowRight, 
  Clock,
  CheckCircle2,
  XCircle,
  Loader2
} from 'lucide-react';
import { getTraces, Trace, formatDuration } from '@/utils/api';

interface PipelineData {
  id: string;
  total: number;
  completed: number;
  failed: number;
  running: number;
  avgDuration: number;
  lastRun: string | null;
}

export default function PipelinesPage() {
  const [pipelines, setPipelines] = useState<PipelineData[]>([]);
  const [recentTraces, setRecentTraces] = useState<Trace[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadData() {
      try {
        const response = await getTraces({ limit: 100 });
        const traces = response.results || [];
        setRecentTraces(traces.slice(0, 5));
        
        // Group by pipeline
        const pipelineMap = new Map<string, Trace[]>();
        traces.forEach(trace => {
          const list = pipelineMap.get(trace.pipeline_id) || [];
          list.push(trace);
          pipelineMap.set(trace.pipeline_id, list);
        });
        
        const pipelineData: PipelineData[] = [];
        pipelineMap.forEach((traceList, id) => {
          const completed = traceList.filter(t => t.status === 'completed').length;
          const failed = traceList.filter(t => t.status === 'failed').length;
          const running = traceList.filter(t => t.status === 'running').length;
          
          const durations = traceList
            .filter(t => t.started_at && t.ended_at)
            .map(t => new Date(t.ended_at!).getTime() - new Date(t.started_at).getTime());
          
          const sorted = [...traceList].sort(
            (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
          );
          
          pipelineData.push({
            id,
            total: traceList.length,
            completed,
            failed,
            running,
            avgDuration: durations.length > 0 
              ? durations.reduce((a, b) => a + b, 0) / durations.length 
              : 0,
            lastRun: sorted[0]?.started_at || null,
          });
        });
        
        pipelineData.sort((a, b) => b.total - a.total);
        setPipelines(pipelineData);
      } catch (err) {
        console.error('Failed to load:', err);
      } finally {
        setLoading(false);
      }
    }
    
    loadData();
    const interval = setInterval(loadData, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-6 h-6 animate-spin text-[var(--text-tertiary)]" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl animate-fade-in">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-[var(--text-primary)]">Pipelines</h1>
        <p className="text-sm text-[var(--text-tertiary)] mt-1">
          Monitor and analyze your decision pipelines
        </p>
      </div>

      {/* Stats Row */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <StatCard
          label="Total Pipelines"
          value={pipelines.length}
        />
        <StatCard
          label="Total Runs"
          value={pipelines.reduce((acc, p) => acc + p.total, 0)}
        />
        <StatCard
          label="Completed"
          value={pipelines.reduce((acc, p) => acc + p.completed, 0)}
          accent="success"
        />
        <StatCard
          label="Failed"
          value={pipelines.reduce((acc, p) => acc + p.failed, 0)}
          accent="error"
        />
      </div>

      {/* Pipelines Grid */}
      {pipelines.length === 0 ? (
        <div className="card p-12 text-center">
          <Activity className="w-10 h-10 mx-auto mb-3 text-[var(--text-tertiary)]" />
          <p className="text-[var(--text-secondary)]">No pipelines yet</p>
          <p className="text-sm text-[var(--text-tertiary)] mt-1">
            Run a pipeline to see it here
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
          {pipelines.map(pipeline => (
            <PipelineCard key={pipeline.id} pipeline={pipeline} />
          ))}
        </div>
      )}

      {/* Recent Traces */}
      {recentTraces.length > 0 && (
        <div className="card">
          <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-primary)]">
            <h2 className="text-sm font-medium text-[var(--text-primary)]">Recent Traces</h2>
            <Link href="/traces" className="text-xs text-[var(--accent)] hover:underline">
              View all
            </Link>
          </div>
          <div className="divide-y divide-[var(--border-primary)]">
            {recentTraces.map(trace => (
              <TraceRow key={trace.trace_id} trace={trace} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value, accent }: { label: string; value: number; accent?: 'success' | 'error' }) {
  const valueColor = accent === 'success' 
    ? 'text-[var(--success)]' 
    : accent === 'error' 
    ? 'text-[var(--error)]' 
    : 'text-[var(--text-primary)]';
    
  return (
    <div className="card p-4">
      <p className="text-xs text-[var(--text-tertiary)] mb-1">{label}</p>
      <p className={`text-2xl font-semibold ${valueColor}`}>{value}</p>
    </div>
  );
}

function PipelineCard({ pipeline }: { pipeline: PipelineData }) {
  const successRate = pipeline.total > 0 
    ? ((pipeline.completed / pipeline.total) * 100).toFixed(0)
    : 0;
    
  return (
    <Link 
      href={`/pipelines/${encodeURIComponent(pipeline.id)}`}
      className="card card-hover p-4 block group"
    >
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="text-sm font-medium text-[var(--text-primary)] group-hover:text-[var(--accent)] transition-colors">
            {pipeline.id}
          </h3>
          <p className="text-xs text-[var(--text-tertiary)] mt-0.5">
            {pipeline.total} runs
          </p>
        </div>
        <ArrowRight className="w-4 h-4 text-[var(--text-tertiary)] group-hover:text-[var(--accent)] transition-colors" />
      </div>
      
      <div className="flex items-center gap-4 text-xs">
        <div className="flex items-center gap-1.5">
          <CheckCircle2 className="w-3.5 h-3.5 text-[var(--success)]" />
          <span className="text-[var(--text-secondary)]">{pipeline.completed}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <XCircle className="w-3.5 h-3.5 text-[var(--error)]" />
          <span className="text-[var(--text-secondary)]">{pipeline.failed}</span>
        </div>
        {pipeline.running > 0 && (
          <div className="flex items-center gap-1.5">
            <Loader2 className="w-3.5 h-3.5 text-[var(--accent)] animate-spin" />
            <span className="text-[var(--text-secondary)]">{pipeline.running}</span>
          </div>
        )}
        <div className="flex items-center gap-1.5 ml-auto">
          <Clock className="w-3.5 h-3.5 text-[var(--text-tertiary)]" />
          <span className="text-[var(--text-tertiary)]">
            {formatDurationMs(pipeline.avgDuration)}
          </span>
        </div>
      </div>
      
      {/* Progress bar */}
      <div className="mt-3 h-1 bg-[var(--bg-tertiary)] rounded-full overflow-hidden flex">
        {pipeline.completed > 0 && (
          <div 
            className="bg-[var(--success)] h-full"
            style={{ width: `${(pipeline.completed / pipeline.total) * 100}%` }}
          />
        )}
        {pipeline.running > 0 && (
          <div 
            className="bg-[var(--accent)] h-full"
            style={{ width: `${(pipeline.running / pipeline.total) * 100}%` }}
          />
        )}
        {pipeline.failed > 0 && (
          <div 
            className="bg-[var(--error)] h-full"
            style={{ width: `${(pipeline.failed / pipeline.total) * 100}%` }}
          />
        )}
      </div>
    </Link>
  );
}

function TraceRow({ trace }: { trace: Trace }) {
  const statusIcon = trace.status === 'completed' ? (
    <CheckCircle2 className="w-3.5 h-3.5 text-[var(--success)]" />
  ) : trace.status === 'failed' ? (
    <XCircle className="w-3.5 h-3.5 text-[var(--error)]" />
  ) : (
    <Loader2 className="w-3.5 h-3.5 text-[var(--accent)] animate-spin" />
  );
  
  return (
    <Link 
      href={`/traces/${trace.trace_id}`}
      className="flex items-center justify-between px-4 py-3 hover:bg-[var(--bg-tertiary)] transition-colors"
    >
      <div className="flex items-center gap-3">
        {statusIcon}
        <span className="text-sm text-[var(--text-primary)]">{trace.pipeline_id}</span>
        <span className="text-xs text-[var(--text-tertiary)] font-mono">
          {trace.trace_id.slice(0, 8)}
        </span>
      </div>
      <span className="text-xs text-[var(--text-tertiary)]">
        {formatDuration(trace.started_at, trace.ended_at)}
      </span>
    </Link>
  );
}

function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}
