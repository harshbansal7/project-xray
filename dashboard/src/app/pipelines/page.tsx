'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { ListTree, Activity, Clock, CheckCircle, XCircle, TrendingUp } from 'lucide-react';
import { getTraces, Trace, formatDuration } from '@/utils/api';

interface PipelineStats {
  pipeline_id: string;
  total: number;
  completed: number;
  failed: number;
  running: number;
  successRate: number;
  avgDuration: number;
  lastRun: string;
}

export default function PipelinesPage() {
  const [pipelines, setPipelines] = useState<PipelineStats[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadPipelines() {
      try {
        setLoading(true);
        const response = await getTraces({ limit: 200 });
        const traces = response.results || [];
        
        // Group by pipeline
        const pipelineMap = new Map<string, Trace[]>();
        traces.forEach(trace => {
          const list = pipelineMap.get(trace.pipeline_id) || [];
          list.push(trace);
          pipelineMap.set(trace.pipeline_id, list);
        });
        
        // Calculate stats for each pipeline
        const stats: PipelineStats[] = [];
        pipelineMap.forEach((traceList, pipelineId) => {
          const completed = traceList.filter(t => t.status === 'completed').length;
          const failed = traceList.filter(t => t.status === 'failed').length;
          const running = traceList.filter(t => t.status === 'running').length;
          
          const durations = traceList
            .filter(t => t.started_at && t.ended_at)
            .map(t => new Date(t.ended_at!).getTime() - new Date(t.started_at).getTime());
          const avgDuration = durations.length > 0 
            ? durations.reduce((a, b) => a + b, 0) / durations.length 
            : 0;
          
          const sorted = [...traceList].sort(
            (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
          );
          
          stats.push({
            pipeline_id: pipelineId,
            total: traceList.length,
            completed,
            failed,
            running,
            successRate: traceList.length > 0 ? (completed / traceList.length) * 100 : 0,
            avgDuration,
            lastRun: sorted[0]?.started_at || '',
          });
        });
        
        // Sort by total runs descending
        stats.sort((a, b) => b.total - a.total);
        setPipelines(stats);
      } catch (err) {
        console.error('Failed to load pipelines:', err);
      } finally {
        setLoading(false);
      }
    }
    
    loadPipelines();
  }, []);

  return (
    <div className="p-8 animate-fade-in">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white">Pipelines</h1>
        <p className="text-gray-400 mt-1">Overview of all registered pipelines</p>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-500" />
        </div>
      ) : pipelines.length === 0 ? (
        <div className="text-center py-12 bg-gray-900 border border-gray-800 rounded-xl">
          <ListTree className="w-12 h-12 mx-auto mb-3 text-gray-600" />
          <p className="text-gray-400">No pipelines found</p>
          <p className="text-sm text-gray-500 mt-1">Run a pipeline to see it here</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {pipelines.map(pipeline => (
            <Link
              key={pipeline.pipeline_id}
              href={`/traces?pipeline_id=${encodeURIComponent(pipeline.pipeline_id)}`}
              className="bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-violet-500/50 transition-all hover:shadow-lg hover:shadow-violet-500/10"
            >
              {/* Pipeline Name */}
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-violet-500/10 rounded-lg">
                  <ListTree className="w-5 h-5 text-violet-400" />
                </div>
                <h3 className="text-lg font-semibold text-white">{pipeline.pipeline_id}</h3>
              </div>

              {/* Stats Grid */}
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div>
                  <p className="text-xs text-gray-500">Total Runs</p>
                  <p className="text-xl font-bold text-white">{pipeline.total}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Success Rate</p>
                  <p className={`text-xl font-bold ${
                    pipeline.successRate >= 90 ? 'text-green-500' :
                    pipeline.successRate >= 70 ? 'text-yellow-500' : 'text-red-500'
                  }`}>
                    {pipeline.successRate.toFixed(0)}%
                  </p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Avg Duration</p>
                  <p className="text-lg font-semibold text-white">
                    {formatDurationMs(pipeline.avgDuration)}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Running</p>
                  <p className="text-lg font-semibold text-blue-400">{pipeline.running}</p>
                </div>
              </div>

              {/* Status Bar */}
              {pipeline.total > 0 && (
                <div className="h-2 bg-gray-800 rounded-full overflow-hidden flex">
                  <div 
                    className="bg-green-500 h-full"
                    style={{ width: `${(pipeline.completed / pipeline.total) * 100}%` }}
                  />
                  <div 
                    className="bg-blue-500 h-full"
                    style={{ width: `${(pipeline.running / pipeline.total) * 100}%` }}
                  />
                  <div 
                    className="bg-red-500 h-full"
                    style={{ width: `${(pipeline.failed / pipeline.total) * 100}%` }}
                  />
                </div>
              )}

              {/* Last Run */}
              {pipeline.lastRun && (
                <p className="text-xs text-gray-500 mt-3">
                  Last run: {new Date(pipeline.lastRun).toLocaleString()}
                </p>
              )}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}
