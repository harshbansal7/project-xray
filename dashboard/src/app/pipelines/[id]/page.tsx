'use client';

import { useEffect, useState, useMemo } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { 
  ArrowLeft, 
  CheckCircle2, 
  XCircle, 
  Loader2, 
  TrendingUp, 
  Filter, 
  Zap, 
  Tag, 
  Search, 
  ChevronRight,
  Clock,
  BarChart3,
  Hash,
  ArrowDownRight,
  Layers,
  ChevronDown,
  ChevronUp,
  Box,
  PieChart
} from 'lucide-react';
import { getTraces, getEvents, getDecisionsByEvent, queryDecisions, Trace, Event, Decision, formatDuration } from '@/utils/api';
import { format } from 'date-fns';
import { OutcomeChart, StepFlowDiagram, ReasonCodeBreakdown, StepData, DecisionFunnel } from '@/components/charts';
import { getOutcomeColor } from '@/utils/colors';

interface StepStats {
  type: string;
  count: number;
  avgReduction: number;
  avgDuration: number;
  totalInputs: number;
  totalOutputs: number;
}

interface DecisionStats {
  outcomes: Record<string, number>;
  reasonCodes: Record<string, number>;
  totalDecisions: number;
}

export default function PipelineDetailPage() {
  const params = useParams();
  const pipelineId = decodeURIComponent(params.id as string);
  
  const [traces, setTraces] = useState<Trace[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [stepStats, setStepStats] = useState<StepStats[]>([]);
  const [decisionStats, setDecisionStats] = useState<DecisionStats>({ 
    outcomes: {}, 
    reasonCodes: {}, 
    totalDecisions: 0 
  });
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'overview' | 'traces' | 'steps' | 'decisions'>('overview');
  
  // Filtering & Detail States
  const [tagFilter, setTagFilter] = useState('');
  const [metaFilter, setMetaFilter] = useState(''); // key:value
  const [selectedStep, setSelectedStep] = useState<StepStats | null>(null);
  const [selectedStepDecisions, setSelectedStepDecisions] = useState<Decision[]>([]);
  const [selectedStepStats, setSelectedStepStats] = useState<DecisionStats | null>(null);
  
  // Decisions tab state - step-driven
  const [selectedDecisionStepType, setSelectedDecisionStepType] = useState<string | null>(null);
  const [stepDecisions, setStepDecisions] = useState<Decision[]>([]);
  const [stepDecisionsLoading, setStepDecisionsLoading] = useState(false);
  const [decisionSearch, setDecisionSearch] = useState('');
  const [decisionOutcomeFilter, setDecisionOutcomeFilter] = useState<string>('all');
  const [expandedDecisionId, setExpandedDecisionId] = useState<string | null>(null);

  useEffect(() => {
    if (selectedStep) {
      const fetchStepDecisions = async () => {
        try {
          const res = await queryDecisions({
            pipeline_id: pipelineId,
            step_type: selectedStep.type,
            limit: 100
          });
          const decisions = res.decisions || [];
          setSelectedStepDecisions(decisions);
          
          const outcomes: Record<string, number> = {};
          const reasons: Record<string, number> = {};
          decisions.forEach(d => {
            outcomes[d.outcome] = (outcomes[d.outcome] || 0) + 1;
            if (d.reason_code) {
              reasons[d.reason_code] = (reasons[d.reason_code] || 0) + 1;
            }
          });
          
          setSelectedStepStats({
            outcomes,
            reasonCodes: reasons,
            totalDecisions: decisions.length
          });
        } catch (err) {
          console.error('Failed to fetch step decisions:', err);
        }
      };
      
      fetchStepDecisions();
    } else {
      setSelectedStepDecisions([]);
      setSelectedStepStats(null);
    }
  }, [selectedStep, pipelineId]);

  // Load decisions for selected step type in decisions tab
  useEffect(() => {
    if (activeTab === 'decisions' && selectedDecisionStepType) {
      const loadStepDecisions = async () => {
        setStepDecisionsLoading(true);
        setStepDecisions([]);
        try {
          const res = await queryDecisions({
            pipeline_id: pipelineId,
            step_type: selectedDecisionStepType,
            limit: 500
          });
          setStepDecisions(res.decisions || []);
        } catch (err) {
          console.error('Failed to load step decisions:', err);
        } finally {
          setStepDecisionsLoading(false);
        }
      };
      loadStepDecisions();
    }
  }, [activeTab, pipelineId, selectedDecisionStepType]);

  // Reset decisions tab state when switching away
  useEffect(() => {
    if (activeTab !== 'decisions') {
      setSelectedDecisionStepType(null);
      setStepDecisions([]);
      setDecisionSearch('');
      setDecisionOutcomeFilter('all');
    }
  }, [activeTab]);

  const loadData = async () => {
    try {
      const filters: any = { pipeline_id: pipelineId, limit: 50 };
      if (tagFilter) filters.tags = tagFilter.split(',').map(t => t.trim());
      
      if (metaFilter && metaFilter.includes(':')) {
        const [key, val] = metaFilter.split(':');
        filters.metadata = { [key.trim()]: val.trim() };
      }

      const [traceRes, eventRes] = await Promise.all([
        getTraces(filters),
        getEvents({ pipeline_id: pipelineId, limit: 200 })
      ]);
      
      const loadedTraces = traceRes.results || [];
      const loadedEvents = eventRes.events || [];
      
      setTraces(loadedTraces);
      setEvents(loadedEvents);
      
      // Group events by step type for stats
      const stepMap = new Map<string, Event[]>();
      loadedEvents.forEach(evt => {
        const list = stepMap.get(evt.step_type) || [];
        list.push(evt);
        stepMap.set(evt.step_type, list);
      });

      const stats: StepStats[] = [];
      stepMap.forEach((evts, stepType) => {
        const reductions = evts
          .filter(e => e.input_count && e.output_count && e.input_count > 0)
          .map(e => 1 - (e.output_count! / e.input_count!));
        
        const durations = evts
          .filter(e => e.started_at && e.ended_at)
          .map(e => new Date(e.ended_at!).getTime() - new Date(e.started_at).getTime());
        
        const totalInputs = evts.reduce((s, e) => s + (e.input_count || 0), 0);
        const totalOutputs = evts.reduce((s, e) => s + (e.output_count || 0), 0);
        
        stats.push({
          type: stepType,
          count: evts.length,
          avgReduction: reductions.length > 0
            ? reductions.reduce((a, b) => a + b, 0) / reductions.length
            : 0,
          avgDuration: durations.length > 0
            ? durations.reduce((a, b) => a + b, 0) / durations.length
            : 0,
          totalInputs,
          totalOutputs,
        });
      });
      
      setStepStats(stats);
      
      // Load decision stats from first few traces
      await loadDecisionStats(loadedTraces.slice(0, 5), loadedEvents);
      
    } catch (err) {
      console.error('Failed to load:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [pipelineId, tagFilter, metaFilter]);

  async function loadDecisionStats(tracesToLoad: Trace[], eventsToUse: Event[]) {
    const outcomes: Record<string, number> = {};
    const reasonCodes: Record<string, number> = {};
    let totalDecisions = 0;
    
    // Get decisions from first events of each trace
    for (const trace of tracesToLoad.slice(0, 3)) {
      const traceEvents = eventsToUse.filter(e => e.trace_id === trace.trace_id);
      
      for (const evt of traceEvents.slice(0, 3)) {
        try {
          const decPage = await getDecisionsByEvent(trace.trace_id, evt.event_id);
          const decisions = decPage.decisions || [];
          
          for (const dec of decisions) {
            outcomes[dec.outcome] = (outcomes[dec.outcome] || 0) + 1;
            if (dec.reason_code) {
              reasonCodes[dec.reason_code] = (reasonCodes[dec.reason_code] || 0) + 1;
            }
            totalDecisions++;
          }
        } catch {
          // Silently continue if decisions can't be loaded
        }
      }
    }
    
    setDecisionStats({ outcomes, reasonCodes, totalDecisions });
  }

  const completed = traces.filter(t => t.status === 'completed').length;
  const failed = traces.filter(t => t.status === 'failed').length;
  const running = traces.filter(t => t.status === 'running').length;
  const successRate = traces.length > 0 ? ((completed / traces.length) * 100).toFixed(0) : '0';
  
  // Compute average duration
  const durations = traces
    .filter(t => t.started_at && t.ended_at)
    .map(t => new Date(t.ended_at!).getTime() - new Date(t.started_at).getTime());
  const avgDuration = durations.length > 0 
    ? durations.reduce((a, b) => a + b, 0) / durations.length 
    : 0;

  // Prepare step flow data
  const stepFlowData: StepData[] = stepStats.map(s => ({
    type: s.type,
    inputCount: Math.round(s.totalInputs / (s.count || 1)),
    outputCount: Math.round(s.totalOutputs / (s.count || 1)),
    durationMs: s.avgDuration,
  }));

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="w-6 h-6 animate-spin text-[var(--accent)]" />
          <p className="text-sm text-[var(--text-tertiary)]">Loading pipeline data...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-none px-4 sm:px-6 lg:px-8 2xl:px-10 py-4 sm:py-6 animate-fade-in">
      {/* Header */}
      <div className="mb-6">
        <Link href="/" className="inline-flex items-center gap-1.5 text-sm text-[var(--text-tertiary)] hover:text-[var(--text-primary)] mb-3 transition-colors">
          <ArrowLeft className="w-4 h-4" />
          Pipelines
        </Link>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-[var(--text-primary)]">{pipelineId}</h1>
            <p className="text-sm text-[var(--text-tertiary)] mt-1">
              {traces.length} total runs • {stepStats.length} steps
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span className={`px-2.5 py-1 rounded-full text-xs font-medium ${
              running > 0 
                ? 'bg-[var(--accent)]/10 text-[var(--accent)]' 
                : 'bg-[var(--bg-tertiary)] text-[var(--text-tertiary)]'
            }`}>
              {running > 0 ? `${running} running` : 'Idle'}
            </span>
          </div>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <MetricCard 
          label="Total Runs" 
          value={traces.length} 
          icon={<Zap className="w-4 h-4" />}
        />
        <MetricCard 
          label="Success Rate" 
          value={`${successRate}%`}
          subValue={`${completed} / ${traces.length}`}
          icon={<TrendingUp className="w-4 h-4" />}
          accent={parseInt(successRate) >= 80 ? 'success' : parseInt(successRate) >= 50 ? 'warning' : 'error'}
        />
        <MetricCard 
          label="Avg Duration" 
          value={formatDurationMs(avgDuration)}
          icon={<Clock className="w-4 h-4" />}
        />
        <MetricCard 
          label="Total Decisions" 
          value={decisionStats.totalDecisions}
          subValue={`${Object.keys(decisionStats.outcomes).length} outcomes`}
          icon={<Filter className="w-4 h-4" />}
        />
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 bg-[var(--bg-secondary)] rounded-lg border border-[var(--border-primary)] mb-6 w-fit">
        {(['overview', 'traces', 'steps', 'decisions'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab as any)}
            className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTab === tab
                ? 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] shadow-sm'
                : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
            }`}
          >
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      {/* Content */}
      {activeTab === 'overview' && (
        <div className="space-y-6">
          {/* Step Flow Diagram */}
          {stepFlowData.length > 0 && (
            <div className="card p-4">
              <h2 className="text-sm font-medium text-[var(--text-primary)] mb-3">Pipeline Flow</h2>
              <StepFlowDiagram steps={stepFlowData} />
            </div>
          )}

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
            {/* Outcome Distribution */}
            {Object.keys(decisionStats.outcomes).length > 0 && (
              <div className="card p-4">
                <h2 className="text-sm font-medium text-[var(--text-primary)] mb-4">Outcome Distribution</h2>
                <OutcomeChart outcomes={decisionStats.outcomes} variant="donut" />
              </div>
            )}

            {/* Reason Code Breakdown */}
            {Object.keys(decisionStats.reasonCodes).length > 0 && (
              <div className="card p-4">
                <h2 className="text-sm font-medium text-[var(--text-primary)] mb-4">Top Reason Codes</h2>
                <ReasonCodeBreakdown reasonCodes={decisionStats.reasonCodes} />
              </div>
            )}

            {/* Recent Traces */}
            <div className="card">
              <div className="px-4 py-3 border-b border-[var(--border-primary)] flex items-center justify-between">
                <h2 className="text-sm font-medium text-[var(--text-primary)]">Recent Traces</h2>
                <button 
                  onClick={() => setActiveTab('traces')}
                  className="text-xs text-[var(--accent)] hover:underline"
                >
                  View all
                </button>
              </div>
              {traces.length === 0 ? (
                <EmptyState 
                  message="No traces yet"
                  hint="Run your pipeline to see traces here"
                />
              ) : (
                <div className="divide-y divide-[var(--border-primary)]">
                  {traces.slice(0, 5).map(trace => (
                    <Link
                      key={trace.trace_id}
                      href={`/traces/${trace.trace_id}`}
                      className="flex items-center justify-between px-4 py-3 hover:bg-[var(--bg-tertiary)] transition-colors"
                    >
                      <div className="flex items-center gap-3">
                        <StatusIcon status={trace.status} />
                        <span className="text-sm text-[var(--text-secondary)] font-mono">
                          {trace.trace_id.slice(0, 8)}
                        </span>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-xs text-[var(--text-tertiary)]">
                          {formatTimeAgo(trace.started_at)}
                        </span>
                        <span className="text-xs text-[var(--text-secondary)]">
                          {formatDuration(trace.started_at, trace.ended_at)}
                        </span>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </div>

            {/* Step Performance */}
            <div className="card">
              <div className="px-4 py-3 border-b border-[var(--border-primary)] flex items-center justify-between">
                <h2 className="text-sm font-medium text-[var(--text-primary)]">Step Performance</h2>
                <button 
                  onClick={() => setActiveTab('steps')}
                  className="text-xs text-[var(--accent)] hover:underline"
                >
                  View all
                </button>
              </div>
              {stepStats.length === 0 ? (
                <EmptyState 
                  message="No step data"
                  hint="Record events with input/output counts"
                />
              ) : (
                <div className="divide-y divide-[var(--border-primary)]">
                  {stepStats.slice(0, 5).map(step => (
                    <div key={step.type} className="px-4 py-3">
                      <div className="flex items-center justify-between mb-1.5">
                        <span className="text-sm font-medium text-[var(--text-primary)]">{step.type}</span>
                      </div>
                      <div className="flex items-center gap-4 text-xs text-[var(--text-tertiary)]">
                        <span>{step.count} runs</span>
                        {step.avgReduction > 0 && (
                          <span className={step.avgReduction >= 0.8 ? 'text-[var(--error)]' : step.avgReduction >= 0.5 ? 'text-[var(--warning)]' : ''}>
                            {(step.avgReduction * 100).toFixed(0)}% reduction
                          </span>
                        )}
                        <span className="ml-auto flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {formatDurationMs(step.avgDuration)}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'traces' && (
        <div className="space-y-4">
          {/* Filter Bar */}
          <div className="flex flex-wrap gap-4 p-4 bg-[var(--bg-secondary)] rounded-xl border border-[var(--border-primary)] shadow-sm">
            <div className="flex-1 min-w-[200px] relative">
              <Tag className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
              <input 
                type="text" 
                placeholder="Filter by tags (comma separated)..."
                className="w-full pl-10 pr-4 py-2 bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg text-sm placeholder:text-[var(--text-tertiary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/30 focus:border-[var(--accent)]"
                value={tagFilter}
                onChange={(e) => setTagFilter(e.target.value)}
              />
            </div>
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
              <input 
                type="text" 
                placeholder="Metadata filter (e.g. env:prod)..."
                className="w-full pl-10 pr-4 py-2 bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg text-sm placeholder:text-[var(--text-tertiary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/30 focus:border-[var(--accent)]"
                value={metaFilter}
                onChange={(e) => setMetaFilter(e.target.value)}
              />
            </div>
          </div>

          <div className="card overflow-hidden shadow-sm">
            <table className="w-full">
              <thead>
                <tr>
                  <th className="w-12">Status</th>
                  <th>Trace ID</th>
                  <th>Started</th>
                  <th>Duration</th>
                  <th className="text-right">Events</th>
                </tr>
              </thead>
              <tbody>
                {traces.length === 0 ? (
                  <tr>
                    <td colSpan={5}>
                      <EmptyState message="No traces recorded" hint="Run your pipeline or check your filters" />
                    </td>
                  </tr>
                ) : (
                  traces.map(trace => {
                    const traceEvents = events.filter(e => e.trace_id === trace.trace_id);
                    return (
                      <tr key={trace.trace_id}>
                        <td className="text-center">
                          <StatusIcon status={trace.status} />
                        </td>
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
                        <td className="text-right text-[var(--text-tertiary)]">{traceEvents.length}</td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'steps' && (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
          <div className="lg:col-span-2 card overflow-hidden shadow-sm">
            <table className="w-full">
              <thead>
                <tr>
                  <th>Step Name</th>
                  <th>Type</th>
                  <th className="text-right">Runs</th>
                  <th className="text-right">Reduction</th>
                  <th className="text-right">Duration</th>
                  <th className="w-10"></th>
                </tr>
              </thead>
              <tbody>
                {stepStats.length === 0 ? (
                  <tr>
                    <td colSpan={6}>
                      <EmptyState message="No step data" hint="Record events with set_input() and set_output()" />
                    </td>
                  </tr>
                ) : (
                  stepStats.map(step => (
                    <tr 
                      key={step.type}
                      onClick={() => setSelectedStep(step)}
                      className={`cursor-pointer transition-colors ${selectedStep?.type === step.type ? 'bg-[var(--bg-tertiary)]' : ''}`}
                    >
                      <td className="text-[var(--text-primary)] font-medium">
                        <div className="flex items-center gap-2">
                          <Zap className={`w-3.5 h-3.5 ${selectedStep?.type === step.type ? 'text-[var(--accent)]' : 'text-[var(--text-tertiary)]'}`} />
                          <span className="px-2 py-0.5 rounded bg-[var(--bg-tertiary)] text-[10px] font-medium uppercase tracking-wider text-[var(--text-tertiary)] border border-[var(--border-primary)]">
                            {step.type}
                          </span>
                        </div>
                      </td>
                      <td className="text-right text-sm">{step.count}</td>
                      <td className="text-right">
                        {step.avgReduction > 0 ? (
                          <span className={`font-medium text-sm ${step.avgReduction >= 0.8 ? 'text-[var(--error)]' : step.avgReduction >= 0.5 ? 'text-[var(--warning)]' : ''}`}>
                            {(step.avgReduction * 100).toFixed(0)}%
                          </span>
                        ) : <span className="text-[var(--text-tertiary)]">—</span>}
                      </td>
                      <td className="text-right text-xs text-[var(--text-tertiary)]">{formatDurationMs(step.avgDuration)}</td>
                      <td className="text-right px-2">
                        <ChevronRight className={`w-4 h-4 transition-transform ${selectedStep?.type === step.type ? 'translate-x-1 text-[var(--accent)]' : 'text-[var(--text-tertiary)]'}`} />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Step Detail Sidebar */}
          <div className="lg:col-span-1 space-y-4">
            {selectedStep ? (
              <div className="card p-5 sticky top-6 border-[var(--accent)]/30 border-l-4">
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <h3 className="text-lg font-semibold text-[var(--text-primary)]">{selectedStep.type}</h3>
                  </div>
                  <button 
                    onClick={() => setSelectedStep(null)}
                    className="p-1 hover:bg-[var(--bg-tertiary)] rounded-full text-[var(--text-tertiary)] transition-colors"
                  >
                    <XCircle className="w-4 h-4" />
                  </button>
                </div>

                <div className="space-y-6">
                  {selectedStepStats ? (
                    <>
                      <div className="space-y-2">
                        <div>
                          <p className="text-[10px] text-[var(--text-tertiary)] uppercase font-bold mb-2 flex items-center gap-1">
                            <TrendingUp className="w-3 h-3" /> Outcome Distribution
                          </p>
                            <OutcomeChart outcomes={selectedStepStats.outcomes} variant="bar" />
                        </div>

                        {Object.keys(selectedStepStats.reasonCodes).length > 0 && (
                          <div>
                            <p className="pt-4 text-[10px] text-[var(--text-tertiary)] uppercase font-bold mb-2 flex items-center gap-1">
                              <Filter className="w-3 h-3" /> Top Reason Codes
                            </p>
                            <ReasonCodeBreakdown reasonCodes={selectedStepStats.reasonCodes} maxItems={5} />
                          </div>
                        )}
                      </div>
                    </>
                  ) : (
                    <div className="flex flex-col items-center justify-center py-12 text-[var(--text-tertiary)]">
                      <Loader2 className="w-6 h-6 animate-spin mb-2" />
                      <p className="text-xs">Loading decision analytics...</p>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="card p-8 text-center border-dashed flex flex-col items-center justify-center min-h-[400px]">
                <div className="w-12 h-12 bg-[var(--bg-secondary)] rounded-full flex items-center justify-center mb-4 border border-[var(--border-primary)]">
                  <Zap className="w-6 h-6 text-[var(--text-tertiary)]" />
                </div>
                <h3 className="text-sm font-medium text-[var(--text-primary)]">Select a step</h3>
                <p className="text-xs text-[var(--text-tertiary)] mt-2 max-w-[200px] mx-auto leading-relaxed">
                  Click on a pipeline step to reveal detailed conversion metrics and performance insights
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'decisions' && (
        <DecisionsTabContent
          stepStats={stepStats}
          selectedStepType={selectedDecisionStepType}
          setSelectedStepType={setSelectedDecisionStepType}
          stepDecisions={stepDecisions}
          stepDecisionsLoading={stepDecisionsLoading}
          decisionSearch={decisionSearch}
          setDecisionSearch={setDecisionSearch}
          decisionOutcomeFilter={decisionOutcomeFilter}
          setDecisionOutcomeFilter={setDecisionOutcomeFilter}
          expandedDecisionId={expandedDecisionId}
          setExpandedDecisionId={setExpandedDecisionId}
          pipelineId={pipelineId}
        />
      )}
    </div>
  );
}

// Components

function MetricCard({ 
  label, 
  value, 
  subValue,
  icon,
  accent 
}: { 
  label: string; 
  value: string | number; 
  subValue?: string;
  icon?: React.ReactNode;
  accent?: 'success' | 'warning' | 'error';
}) {
  const valueColor = accent === 'success' 
    ? 'text-[var(--success)]' 
    : accent === 'warning'
    ? 'text-[var(--warning)]'
    : accent === 'error' 
    ? 'text-[var(--error)]' 
    : 'text-[var(--text-primary)]';
    
  return (
    <div className="card p-4">
      <div className="flex items-center gap-2 mb-2">
        {icon && <span className="text-[var(--text-tertiary)]">{icon}</span>}
        <p className="text-xs text-[var(--text-tertiary)] uppercase tracking-wide">{label}</p>
      </div>
      <p className={`text-2xl font-semibold ${valueColor}`}>{value}</p>
      {subValue && (
        <p className="text-xs text-[var(--text-tertiary)] mt-0.5">{subValue}</p>
      )}
    </div>
  );
}

function StatusIcon({ status }: { status: string }) {
  if (status === 'completed') return <CheckCircle2 className="w-4 h-4 text-[var(--success)]" />;
  if (status === 'failed') return <XCircle className="w-4 h-4 text-[var(--error)]" />;
  return <Loader2 className="w-4 h-4 text-[var(--accent)] animate-spin" />;
}

function EmptyState({ message, hint }: { message: string; hint?: string }) {
  return (
    <div className="py-8 px-4 text-center">
      <p className="text-sm text-[var(--text-secondary)]">{message}</p>
      {hint && <p className="text-xs text-[var(--text-tertiary)] mt-1">{hint}</p>}
    </div>
  );
}

function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);
  
  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

// Decisions Tab Content Component - Step-Driven
function DecisionsTabContent({
  stepStats,
  selectedStepType,
  setSelectedStepType,
  stepDecisions,
  stepDecisionsLoading,
  decisionSearch,
  setDecisionSearch,
  decisionOutcomeFilter,
  setDecisionOutcomeFilter,
  expandedDecisionId,
  setExpandedDecisionId,
  pipelineId,
}: {
  stepStats: StepStats[];
  selectedStepType: string | null;
  setSelectedStepType: (v: string | null) => void;
  stepDecisions: Decision[];
  stepDecisionsLoading: boolean;
  decisionSearch: string;
  setDecisionSearch: (v: string) => void;
  decisionOutcomeFilter: string;
  setDecisionOutcomeFilter: (v: string) => void;
  expandedDecisionId: string | null;
  setExpandedDecisionId: (v: string | null) => void;
  pipelineId: string;
}) {
  // Compute stats from step decisions
  const stats = useMemo(() => {
    const outcomes: Record<string, number> = {};
    const reasonCodes: Record<string, number> = {};
    
    stepDecisions.forEach(d => {
      outcomes[d.outcome] = (outcomes[d.outcome] || 0) + 1;
      if (d.reason_code) {
        reasonCodes[d.reason_code] = (reasonCodes[d.reason_code] || 0) + 1;
      }
    });
    
    return {
      total: stepDecisions.length,
      outcomes,
      reasonCodes,
      uniqueOutcomes: Object.keys(outcomes).sort(),
    };
  }, [stepDecisions]);

  // Filter decisions
  const filteredDecisions = useMemo(() => {
    return stepDecisions.filter(d => {
      if (decisionOutcomeFilter !== 'all' && d.outcome !== decisionOutcomeFilter) return false;
      if (decisionSearch) {
        const q = decisionSearch.toLowerCase();
        return (
          d.item_id.toLowerCase().includes(q) ||
          d.outcome.toLowerCase().includes(q) ||
          d.reason_code?.toLowerCase().includes(q) ||
          d.reason_detail?.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [stepDecisions, decisionOutcomeFilter, decisionSearch]);

  // No steps with decisions
  if (stepStats.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <Layers className="w-12 h-12 text-[var(--text-tertiary)] mb-4 opacity-50" />
        <h3 className="text-lg font-medium text-[var(--text-primary)] mb-2">No Steps Found</h3>
        <p className="text-sm text-[var(--text-tertiary)] max-w-md">
          No pipeline steps have been recorded yet. Run your pipeline to see step-level decisions.
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-12 gap-6">
      {/* Left Panel: Step Selector */}
      <div className="col-span-12 lg:col-span-4 xl:col-span-3">
        <div className="card p-4 sticky top-6">
          <h3 className="text-sm font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
            <Zap className="w-4 h-4 text-[var(--accent)]" />
            Select Step Type
          </h3>
          <p className="text-xs text-[var(--text-tertiary)] mb-4">
            Decisions are grouped by the step that produced them. Select a step to analyze its decisions.
          </p>
          
          <div className="space-y-2">
            {stepStats.map(step => {
              const isSelected = selectedStepType === step.type;
              return (
                <button
                  key={step.type}
                  onClick={() => {
                    setSelectedStepType(step.type);
                    setDecisionOutcomeFilter('all');
                    setDecisionSearch('');
                    setExpandedDecisionId(null);
                  }}
                  className={`w-full text-left p-3 rounded-lg border transition-all ${
                    isSelected
                      ? 'bg-[var(--accent)]/10 border-[var(--accent)] ring-1 ring-[var(--accent)]/30'
                      : 'bg-[var(--bg-primary)] border-[var(--border-primary)] hover:border-[var(--border-secondary)] hover:bg-[var(--bg-tertiary)]'
                  }`}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className={`text-xs font-bold uppercase tracking-wider ${isSelected ? 'text-[var(--accent)]' : 'text-[var(--text-tertiary)]'}`}>
                      {step.type}
                    </span>
                    <ChevronRight className={`w-4 h-4 transition-transform ${isSelected ? 'text-[var(--accent)] translate-x-0.5' : 'text-[var(--text-tertiary)]'}`} />
                  </div>
                  <div className="flex items-center gap-3 text-xs text-[var(--text-secondary)]">
                    <span>{step.count} executions</span>
                    {step.avgReduction > 0 && (
                      <span className="text-[var(--accent)]">
                        {(step.avgReduction * 100).toFixed(0)}% reduction
                      </span>
                    )}
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {/* Right Panel: Step Decision Analysis */}
      <div className="col-span-12 lg:col-span-8 xl:col-span-9">
        {!selectedStepType ? (
          <div className="card p-8 flex flex-col items-center justify-center text-center min-h-[400px]">
            <ArrowLeft className="w-10 h-10 text-[var(--text-tertiary)] mb-4 opacity-50 rotate-180 lg:rotate-0" />
            <h3 className="text-lg font-medium text-[var(--text-primary)] mb-2">Select a Step Type</h3>
            <p className="text-sm text-[var(--text-tertiary)] max-w-sm">
              Choose a step type from the left to view its decision analytics and explore individual decisions.
            </p>
          </div>
        ) : stepDecisionsLoading ? (
          <div className="card p-8 flex flex-col items-center justify-center min-h-[400px]">
            <Loader2 className="w-8 h-8 text-[var(--accent)] animate-spin mb-3" />
            <p className="text-sm text-[var(--text-tertiary)]">Loading decisions for {selectedStepType}...</p>
          </div>
        ) : stepDecisions.length === 0 ? (
          <div className="card p-8 flex flex-col items-center justify-center text-center min-h-[400px]">
            <BarChart3 className="w-10 h-10 text-[var(--text-tertiary)] mb-4 opacity-50" />
            <h3 className="text-lg font-medium text-[var(--text-primary)] mb-2">No Decisions for "{selectedStepType}"</h3>
            <p className="text-sm text-[var(--text-tertiary)] max-w-sm">
              This step hasn't recorded any decisions yet. Use{' '}
              <code className="px-1.5 py-0.5 bg-[var(--bg-tertiary)] rounded text-xs">event.record_decision()</code>{' '}
              in your pipeline code.
            </p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Step Header */}
            <div className="card p-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold text-[var(--text-primary)] flex items-center gap-2">
                    <Zap className="w-5 h-5 text-[var(--accent)]" />
                    {selectedStepType}
                  </h3>
                  <p className="text-sm text-[var(--text-tertiary)] mt-1">
                    {stats.total.toLocaleString()} decisions · {stats.uniqueOutcomes.length} outcome types · {Object.keys(stats.reasonCodes).length} reason codes
                  </p>
                </div>
                <button
                  onClick={() => setSelectedStepType(null)}
                  className="text-xs text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] flex items-center gap-1"
                >
                  <ArrowLeft className="w-3 h-3" />
                  Back to steps
                </button>
              </div>
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="card p-4">
                <div className="flex items-center gap-2 mb-1">
                  <BarChart3 className="w-3.5 h-3.5 text-[var(--accent)]" />
                  <span className="text-[10px] text-[var(--text-tertiary)] uppercase tracking-wider">Total</span>
                </div>
                <p className="text-xl font-semibold text-[var(--text-primary)]">{stats.total.toLocaleString()}</p>
              </div>
              {stats.uniqueOutcomes.slice(0, 3).map(outcome => {
                const color = getOutcomeColor(outcome);
                const count = stats.outcomes[outcome];
                const pct = ((count / stats.total) * 100).toFixed(1);
                return (
                  <div key={outcome} className="card p-4">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
                      <span className="text-[10px] text-[var(--text-tertiary)] uppercase tracking-wider capitalize">{outcome}</span>
                    </div>
                    <p className="text-xl font-semibold text-[var(--text-primary)]">{count.toLocaleString()}</p>
                    <p className="text-xs text-[var(--text-tertiary)]">{pct}%</p>
                  </div>
                );
              })}
            </div>

            {/* Charts Row */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="card p-5">
                <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                  <PieChart className="w-4 h-4 text-[var(--accent)]" />
                  Outcome Distribution
                </h4>
                {Object.keys(stats.outcomes).length > 0 ? (
                  <OutcomeChart outcomes={stats.outcomes} variant="bar" />
                ) : (
                  <EmptyState message="No outcomes" />
                )}
              </div>
              
              <div className="card p-5">
                <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                  <Tag className="w-4 h-4 text-[var(--accent)]" />
                  Top Reason Codes
                </h4>
                {Object.keys(stats.reasonCodes).length > 0 ? (
                  <ReasonCodeBreakdown reasonCodes={stats.reasonCodes} maxItems={6} />
                ) : (
                  <EmptyState message="No reason codes" hint="Add reason_code to decisions" />
                )}
              </div>
            </div>

            {/* Decision Explorer */}
            <div className="card p-5">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-4">
                <h4 className="text-sm font-semibold text-[var(--text-primary)] flex items-center gap-2">
                  <Filter className="w-4 h-4 text-[var(--accent)]" />
                  Decisions
                  <span className="text-xs font-normal text-[var(--text-tertiary)] bg-[var(--bg-tertiary)] px-2 py-0.5 rounded-md">
                    {filteredDecisions.length} of {stats.total}
                  </span>
                </h4>
              </div>

              {/* Filters */}
              <div className="flex flex-wrap gap-3 mb-4">
                <div className="relative flex-1 min-w-[200px]">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
                  <input
                    type="text"
                    placeholder="Search item ID, reason code..."
                    value={decisionSearch}
                    onChange={(e) => setDecisionSearch(e.target.value)}
                    className="w-full pl-10 pr-4 py-2 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg text-sm placeholder:text-[var(--text-tertiary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/30 focus:border-[var(--accent)]"
                  />
                </div>

                <div className="flex flex-wrap gap-1.5">
                  <button
                    onClick={() => setDecisionOutcomeFilter('all')}
                    className={`px-3 py-1.5 text-xs font-medium rounded-md border transition-all ${
                      decisionOutcomeFilter === 'all'
                        ? 'bg-[var(--accent)] text-white border-[var(--accent)]'
                        : 'bg-[var(--bg-primary)] text-[var(--text-secondary)] border-[var(--border-primary)] hover:border-[var(--border-secondary)]'
                    }`}
                  >
                    All
                  </button>
                  {stats.uniqueOutcomes.map(outcome => {
                    const color = getOutcomeColor(outcome);
                    const isActive = decisionOutcomeFilter === outcome;
                    return (
                      <button
                        key={outcome}
                        onClick={() => setDecisionOutcomeFilter(outcome)}
                        className="px-3 py-1.5 text-xs font-medium rounded-md border transition-all flex items-center gap-1.5"
                        style={{
                          backgroundColor: isActive ? color : 'var(--bg-primary)',
                          borderColor: isActive ? color : `${color}40`,
                          color: isActive ? 'white' : color,
                        }}
                      >
                        <span className="capitalize">{outcome}</span>
                        <span className={`text-[10px] ${isActive ? 'opacity-80' : 'opacity-60'}`}>
                          {stats.outcomes[outcome]}
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Decision List */}
              <div className="space-y-2 max-h-[400px] overflow-y-auto pr-1">
                {filteredDecisions.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <Search className="w-8 h-8 text-[var(--text-tertiary)] mb-3 opacity-40" />
                    <p className="text-sm text-[var(--text-secondary)]">No decisions match your filters</p>
                  </div>
                ) : (
                  filteredDecisions.slice(0, 100).map((decision) => (
                    <DecisionRow
                      key={decision.decision_id}
                      decision={decision}
                      expanded={expandedDecisionId === decision.decision_id}
                      onToggle={() => setExpandedDecisionId(expandedDecisionId === decision.decision_id ? null : decision.decision_id)}
                    />
                  ))
                )}
                {filteredDecisions.length > 100 && (
                  <div className="text-center py-4 text-xs text-[var(--text-tertiary)]">
                    Showing 100 of {filteredDecisions.length} decisions
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// Decision Row Component
function DecisionRow({ decision, expanded, onToggle }: { decision: Decision; expanded: boolean; onToggle: () => void }) {
  const colorHex = getOutcomeColor(decision.outcome);

  return (
    <div 
      className={`rounded-lg transition-all duration-150 overflow-hidden cursor-pointer ${
        expanded 
          ? 'bg-[var(--bg-secondary)]' 
          : 'bg-[var(--bg-primary)] border border-[var(--border-primary)] hover:border-[var(--border-secondary)]'
      }`}
      style={expanded ? { boxShadow: `0 0 0 2px ${colorHex}30` } : {}}
      onClick={onToggle}
    >
      <div className="p-3 flex items-center gap-3">
        {/* Left: Outcome badge */}
        <span 
          className="px-2 py-1 rounded text-[10px] font-bold uppercase tracking-wider flex-shrink-0"
          style={{ backgroundColor: `${colorHex}15`, color: colorHex }}
        >
          {decision.outcome}
        </span>

        {/* Middle: Item ID and reason code */}
        <div className="flex-1 min-w-0 flex items-center gap-3">
          <div className="flex items-center gap-1.5 min-w-0">
            <Hash className="w-3 h-3 text-[var(--text-tertiary)] flex-shrink-0" />
            <span className="font-mono text-sm text-[var(--text-primary)] truncate" title={decision.item_id}>
              {decision.item_id}
            </span>
          </div>
          {decision.reason_code && (
            <span className="flex items-center gap-1 text-xs text-[var(--text-secondary)] flex-shrink-0 hidden sm:flex">
              <Tag className="w-3 h-3" />
              <span className="font-mono">{decision.reason_code}</span>
            </span>
          )}
        </div>

        {/* Right: Timestamp and expand icon */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {decision.timestamp && (
            <span className="text-[10px] font-mono text-[var(--text-tertiary)] hidden md:block">
              {formatTimeAgo(decision.timestamp)}
            </span>
          )}
          {expanded ? <ChevronUp className="w-4 h-4 text-[var(--text-tertiary)]" /> : <ChevronDown className="w-4 h-4 text-[var(--text-tertiary)]" />}
        </div>
      </div>

      {/* Expanded Content */}
      {expanded && (
        <div className="px-3 pb-3 pt-0 border-t border-[var(--border-primary)]" onClick={e => e.stopPropagation()}>
          {decision.reason_detail && (
            <div className="mt-3">
              <p className="text-[10px] uppercase tracking-wider text-[var(--text-tertiary)] font-semibold mb-1.5">Reason Detail</p>
              <p className="text-sm text-[var(--text-secondary)] bg-[var(--bg-tertiary)] p-2.5 rounded-md leading-relaxed">
                {decision.reason_detail}
              </p>
            </div>
          )}

          {decision.scores && Object.keys(decision.scores).length > 0 && (
            <div className="mt-3">
              <p className="text-[10px] uppercase tracking-wider text-[var(--text-tertiary)] font-semibold mb-1.5">Scores</p>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {Object.entries(decision.scores).map(([key, val]) => (
                  <div key={key} className="flex justify-between text-xs bg-[var(--bg-tertiary)] p-2 rounded-md">
                    <span className="text-[var(--text-secondary)]">{key}</span>
                    <span className="font-mono text-[var(--accent)] font-medium">
                      {typeof val === 'number' ? val.toFixed(2) : val}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {decision.item_snapshot && (
            <div className="mt-3">
              <p className="text-[10px] uppercase tracking-wider text-[var(--text-tertiary)] font-semibold mb-1.5 flex items-center gap-1">
                <Box className="w-3 h-3" /> Item Snapshot
              </p>
              <div className="bg-[var(--bg-tertiary)] rounded-md overflow-hidden">
                <pre className="p-2.5 text-[10px] leading-relaxed overflow-x-auto text-[var(--text-secondary)] font-mono max-h-32">
                  {JSON.stringify(decision.item_snapshot, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
