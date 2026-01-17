import React, { useState, useMemo } from 'react';
import { 
  Search, 
  Filter,
  ChevronDown,
  ChevronUp,
  Box,
  Circle
} from 'lucide-react';
import { Decision } from '@/utils/api';
import { getOutcomeColor } from '@/utils/colors';

interface DecisionStreamProps {
  decisions: Decision[];
  className?: string;
}

export default function DecisionStream({ decisions, className = '' }: DecisionStreamProps) {
  const [activeFilter, setActiveFilter] = useState<string>('all');
  const [search, setSearch] = useState('');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Derive unique outcomes from data for dynamic filtering
  const uniqueOutcomes = useMemo(() => {
    const outcomes = new Set<string>();
    decisions.forEach(d => outcomes.add(d.outcome));
    return Array.from(outcomes).sort();
  }, [decisions]);

  const filteredDecisions = useMemo(() => {
    return decisions.filter(d => {
      // Filter by status
      if (activeFilter !== 'all' && d.outcome !== activeFilter) return false;
      
      // Filter by search
      if (search) {
        const query = search.toLowerCase();
        // Prioritize core fields for search
        return (
          d.item_id.toLowerCase().includes(query) ||
          d.reason_code?.toLowerCase().includes(query) ||
          d.outcome.toLowerCase().includes(query) ||
          d.reason_detail?.toLowerCase().includes(query)
        );
      }
      return true;
    });
  }, [decisions, activeFilter, search]);

  const stats = useMemo(() => {
    const counts: Record<string, number> = { total: decisions.length };
    decisions.forEach(d => {
      counts[d.outcome] = (counts[d.outcome] || 0) + 1;
    });
    return counts;
  }, [decisions]);

  return (
    <div className={`flex flex-col h-full ${className}`}>
      {/* Header & Controls */}
      <div className="flex flex-col gap-4 mb-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-[var(--text-primary)] flex items-center gap-2">
            <Filter className="w-5 h-5" />
            Decision Stream
          </h3>
          <span className="text-sm text-[var(--text-tertiary)] bg-[var(--bg-secondary)] px-2 py-1 rounded-full border border-[var(--border-primary)]">
            {stats.total} Decisions
          </span>
        </div>

        {/* Filter Bar - Dynamic */}
        <div className="flex flex-wrap gap-2 p-1 bg-[var(--bg-secondary)] rounded-lg border border-[var(--border-primary)] w-fit">
          <button
            onClick={() => setActiveFilter('all')}
            className={`px-3 py-1.5 text-sm font-medium rounded-md transition-all ${
              activeFilter === 'all'
                ? 'bg-[var(--bg-primary)] text-[var(--text-primary)] shadow-sm'
                : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
            }`}
          >
            All <span className="ml-1 opacity-60 text-xs">{stats.total}</span>
          </button>
          
          {uniqueOutcomes.map((outcome) => {
            const color = getOutcomeColor(outcome);
            return (
              <button
                key={outcome}
                onClick={() => setActiveFilter(outcome)}
                className={`px-3 py-1.5 text-sm font-medium rounded-md transition-all flex items-center gap-2 ${
                  activeFilter === outcome
                    ? 'bg-[var(--bg-primary)] text-[var(--text-primary)] shadow-sm'
                    : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
                }`}
              >
                <span className="capitalize">{outcome}</span>
                <span 
                  className="text-xs px-1.5 py-0.5 rounded-full"
                  style={{ backgroundColor: `${color}20`, color: color }}
                >
                  {stats[outcome]}
                </span>
              </button>
            );
          })}
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
          <input
            type="text"
            placeholder="Search by Item ID, Outcome, or Reason..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg pl-9 pr-4 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
          />
        </div>
      </div>

      {/* Stream List */}
      <div className="flex-1 overflow-y-auto space-y-3 pr-2 min-h-[400px]">
        {filteredDecisions.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-8 text-[var(--text-tertiary)] border border-dashed border-[var(--border-primary)] rounded-xl">
            <Search className="w-8 h-8 mb-2 opacity-50" />
            <p>No decisions match your filter</p>
          </div>
        ) : (
          filteredDecisions.map((decision) => (
            <DecisionCard 
              key={decision.decision_id} 
              decision={decision} 
              expanded={expandedId === decision.decision_id}
              onToggle={() => setExpandedId(expandedId === decision.decision_id ? null : decision.decision_id)}
            />
          ))
        )}
      </div>
    </div>
  );
}

function DecisionCard({ decision, expanded, onToggle }: { decision: Decision; expanded: boolean; onToggle: () => void }) {
  const colorHex = getOutcomeColor(decision.outcome);
  
  // Dynamic generic styles
  const dynamicStyle = {
    color: colorHex,
  };
  
  const statusClass = `mt-0.5 p-1.5 rounded-full flex-shrink-0 bg-opacity-10`;

  const pillStyle = {
    color: colorHex,
    backgroundColor: `${colorHex}15`,
    borderColor: `${colorHex}30`
  };

  return (
    <div 
      className={`group border rounded-lg transition-all duration-200 overflow-hidden ${
        expanded 
          ? 'bg-[var(--bg-secondary)] shadow-lg scale-[1.01]' 
          : 'border-[var(--border-primary)] hover:border-[var(--border-secondary)] bg-[var(--bg-primary)]'
      }`}
      style={expanded ? { borderColor: colorHex } : {}}
    >
      {/* Card Header (Always Visible) */}
      <div 
        className="p-3 flex items-start gap-3 cursor-pointer"
        onClick={onToggle}
      >
        {/* Status Icon - Generic */}
        <div className={statusClass} style={{ backgroundColor: `${colorHex}15` }}>
            <Circle className="w-4 h-4" style={{ color: colorHex, fill: `${colorHex}40` }} />
        </div>

        {/* content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-1">
            <span className="font-mono text-[13px] font-semibold text-[var(--text-primary)] truncate" title={decision.item_id}>
              {decision.item_id}
            </span>
            <span className="text-xs font-mono text-[var(--text-tertiary)] ml-2 flex-shrink-0">
              {decision.timestamp ? new Date(decision.timestamp).toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit', fractionalSecondDigits: 3 }) : ''}
            </span>
          </div>

          <div className="flex items-center gap-2 text-sm">
            <span 
              className="px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border"
              style={pillStyle}
            >
              {decision.outcome}
            </span>
            {decision.reason_code && (
              <span className="text-[var(--text-secondary)] flex items-center gap-1 overflow-hidden">
                • <span className="font-mono text-xs opacity-80 truncate">{decision.reason_code}</span>
              </span>
            )}
          </div>
        </div>
        
        <div className="mt-1 text-[var(--text-tertiary)]">
          {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </div>
      </div>

      {/* Expanded Details */}
      {expanded && (
        <div className="px-4 pb-4 pt-1 border-t border-[var(--border-primary)] bg-[var(--bg-tertiary)]/30">
          {decision.reason_detail && (
            <div className="mb-4 mt-3">
              <p className="text-xs uppercase tracking-wider text-[var(--text-tertiary)] font-bold mb-1">Reason Detail</p>
              <p className="text-sm text-[var(--text-secondary)] bg-[var(--bg-primary)] p-2 rounded border border-[var(--border-primary)]">
                {decision.reason_detail}
              </p>
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            {decision.scores && Object.keys(decision.scores).length > 0 && (
              <div>
                <p className="text-xs uppercase tracking-wider text-[var(--text-tertiary)] font-bold mb-2">Scores & Metrics</p>
                <div className="space-y-1">
                  {Object.entries(decision.scores).map(([key, val]) => (
                    <div key={key} className="flex justify-between text-xs border-b border-[var(--border-primary)]/50 last:border-0 py-1">
                      <span className="text-[var(--text-secondary)]">{key}</span>
                      <span className="font-mono text-[var(--accent)]">{typeof val === 'number' ? val.toFixed(2) : val}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {decision.item_snapshot && (
              <div className="col-span-2">
                <p className="text-xs uppercase tracking-wider text-[var(--text-tertiary)] font-bold mb-2 flex items-center gap-1">
                  <Box className="w-3 h-3" /> Item Snapshot
                </p>
                <div className="bg-[var(--bg-primary)] rounded border border-[var(--border-primary)] overflow-hidden">
                  <pre className="p-2 text-[10px] leading-relaxed overflow-x-auto text-[var(--text-secondary)] font-mono">
                    {JSON.stringify(decision.item_snapshot, null, 2)}
                  </pre>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
