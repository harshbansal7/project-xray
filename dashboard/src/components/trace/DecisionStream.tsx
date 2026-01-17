import React, { useState, useMemo } from 'react';
import { 
  Search, 
  Filter,
  ChevronDown,
  ChevronUp,
  Box,
  Hash,
  Clock,
  Tag
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
      <div className="flex flex-col gap-3 mb-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-[var(--text-primary)] uppercase tracking-wider flex items-center gap-2">
            <Filter className="w-4 h-4 text-[var(--accent)]" />
            Decision Stream
          </h3>
          <span className="text-xs text-[var(--text-tertiary)] bg-[var(--bg-tertiary)] px-2.5 py-1 rounded-md font-mono">
            {stats.total} total
          </span>
        </div>

        {/* Filter Chips */}
        <div className="flex flex-wrap gap-1.5">
          <button
            onClick={() => setActiveFilter('all')}
            className={`px-2.5 py-1 text-xs font-medium rounded-md transition-all border ${
              activeFilter === 'all'
                ? 'bg-[var(--accent)] text-white border-[var(--accent)]'
                : 'text-[var(--text-secondary)] border-[var(--border-primary)] hover:border-[var(--border-secondary)] bg-[var(--bg-primary)]'
            }`}
          >
            All
          </button>
          
          {uniqueOutcomes.map((outcome) => {
            const color = getOutcomeColor(outcome);
            const isActive = activeFilter === outcome;
            return (
              <button
                key={outcome}
                onClick={() => setActiveFilter(outcome)}
                className={`px-2.5 py-1 text-xs font-medium rounded-md transition-all border flex items-center gap-1.5`}
                style={{
                  backgroundColor: isActive ? color : 'var(--bg-primary)',
                  borderColor: isActive ? color : `${color}40`,
                  color: isActive ? 'white' : color,
                }}
              >
                <span className="capitalize">{outcome}</span>
                <span className={`text-[10px] ${isActive ? 'opacity-80' : 'opacity-60'}`}>
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
            className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg pl-10 pr-4 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/30 focus:border-[var(--accent)]"
          />
        </div>
      </div>

      {/* Stream List */}
      <div className="flex-1 overflow-y-auto space-y-2 pr-1 min-h-[400px]">
        {filteredDecisions.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-8 text-[var(--text-tertiary)] border border-dashed border-[var(--border-primary)] rounded-xl bg-[var(--bg-secondary)]/50">
            <Search className="w-6 h-6 mb-2 opacity-40" />
            <p className="text-sm">No decisions match your filter</p>
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

  return (
    <div 
      className={`rounded-lg transition-all duration-200 overflow-hidden cursor-pointer ${
        expanded 
          ? 'bg-[var(--bg-secondary)]' 
          : 'bg-[var(--bg-primary)] border border-[var(--border-primary)] hover:border-[var(--border-secondary)]'
      }`}
      style={expanded ? { 
        boxShadow: `0 0 0 2px ${colorHex}40`,
      } : {}}
      onClick={onToggle}
    >
      {/* Card Header */}
      <div className="p-3">
        <div className="flex items-start justify-between gap-3">
          {/* Left: Main content */}
          <div className="flex-1 min-w-0">
            {/* Item ID with hash icon */}
            <div className="flex items-center gap-1.5 mb-1.5">
              <Hash className="w-3 h-3 text-[var(--text-tertiary)] flex-shrink-0" />
              <span className="font-mono text-sm font-medium text-[var(--text-primary)] truncate" title={decision.item_id}>
                {decision.item_id}
              </span>
            </div>

            {/* Outcome badge and reason code */}
            <div className="flex items-center gap-2 flex-wrap">
              <span 
                className="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider"
                style={{
                  backgroundColor: `${colorHex}15`,
                  color: colorHex,
                }}
              >
                {decision.outcome}
              </span>
              {decision.reason_code && (
                <span className="flex items-center gap-1 text-xs text-[var(--text-secondary)]">
                  <Tag className="w-3 h-3" />
                  <span className="font-mono">{decision.reason_code}</span>
                </span>
              )}
            </div>
          </div>

          {/* Right: Timestamp and expand icon */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {decision.timestamp && (
              <span className="text-[10px] font-mono text-[var(--text-tertiary)] hidden sm:block">
                {new Date(decision.timestamp).toLocaleTimeString([], { 
                  hour12: false, 
                  hour: '2-digit', 
                  minute: '2-digit', 
                  second: '2-digit' 
                })}
              </span>
            )}
            <div className="text-[var(--text-tertiary)]">
              {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
            </div>
          </div>
        </div>
      </div>

      {/* Expanded Details */}
      {expanded && (
        <div 
          className="px-3 pb-3 pt-0 border-t"
          style={{ borderColor: `${colorHex}20` }}
          onClick={(e) => e.stopPropagation()}
        >
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
              <div className="grid grid-cols-2 gap-2">
                {Object.entries(decision.scores).map(([key, val]) => (
                  <div 
                    key={key} 
                    className="flex justify-between text-xs bg-[var(--bg-tertiary)] p-2 rounded-md"
                  >
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
                <pre className="p-2.5 text-[10px] leading-relaxed overflow-x-auto text-[var(--text-secondary)] font-mono max-h-40">
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
