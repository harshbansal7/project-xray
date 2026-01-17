'use client';

import { getOutcomeColor } from '@/utils/colors';

interface ReasonCodeBreakdownProps {
  reasonCodes: Record<string, number>;
  maxItems?: number;
  className?: string;
  onReasonCodeClick?: (code: string) => void;
}

/**
 * Horizontal bar chart showing reason code frequency
 * Displays top reason codes with counts and percentages
 */
export function ReasonCodeBreakdown({ 
  reasonCodes, 
  maxItems = 6,
  className = '',
  onReasonCodeClick
}: ReasonCodeBreakdownProps) {
  const entries = Object.entries(reasonCodes)
    .sort((a, b) => b[1] - a[1])
    .slice(0, maxItems);
  
  const total = Object.values(reasonCodes).reduce((sum, count) => sum + count, 0);
  const maxCount = entries.length > 0 ? entries[0][1] : 0;

  if (entries.length === 0) {
    return (
      <div className={`text-sm text-[var(--text-tertiary)] ${className}`}>
        No reason codes recorded
      </div>
    );
  }

  return (
    <div className={`space-y-2 ${className}`}>
      {entries.map(([code, count]) => (
        <button
          key={code}
          onClick={() => onReasonCodeClick?.(code)}
          className="w-full flex items-center gap-3 p-1 rounded-md hover:bg-[var(--bg-tertiary)] transition-colors text-left group"
        >
          {/* Reason code name */}
          <span className="w-36 text-xs font-mono text-[var(--text-secondary)] truncate group-hover:text-[var(--text-primary)]">
            {code}
          </span>
          
          {/* Bar */}
          <div className="flex-1 h-4 bg-[var(--bg-tertiary)] rounded overflow-hidden">
            <div
              className="h-full rounded bg-[var(--accent)] opacity-60 group-hover:opacity-100 transition-opacity"
              style={{ width: `${(count / maxCount) * 100}%` }}
            />
          </div>
          
          {/* Count */}
          <span className="w-16 text-right text-xs text-[var(--text-tertiary)]">
            {count} ({((count / total) * 100).toFixed(0)}%)
          </span>
        </button>
      ))}
      
      {Object.keys(reasonCodes).length > maxItems && (
        <p className="text-xs text-[var(--text-tertiary)] pl-2">
          +{Object.keys(reasonCodes).length - maxItems} more reason codes
        </p>
      )}
    </div>
  );
}
