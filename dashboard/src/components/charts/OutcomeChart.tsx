'use client';

import { getOutcomeColor } from '@/utils/colors';

interface OutcomeChartProps {
  outcomes: Record<string, number>;
  variant?: 'pie' | 'bar' | 'donut';
  showLabels?: boolean;
  className?: string;
}

/**
 * Dynamic outcome visualization component
 * Supports any outcome values with consistent color generation
 */
export function OutcomeChart({ 
  outcomes, 
  variant = 'donut',
  showLabels = true,
  className = ''
}: OutcomeChartProps) {
  const entries = Object.entries(outcomes).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((sum, [, count]) => sum + count, 0);
  
  if (total === 0) {
    return (
      <div className={`flex items-center justify-center h-32 text-[var(--text-tertiary)] text-sm ${className}`}>
        No outcome data
      </div>
    );
  }

  if (variant === 'bar') {
    return <BarChart entries={entries} total={total} className={className} />;
  }

  return (
    <div className={`flex items-center gap-6 ${className}`}>
      <DonutChart entries={entries} total={total} />
      {showLabels && <Legend entries={entries} total={total} />}
    </div>
  );
}

interface ChartEntry {
  entries: [string, number][];
  total: number;
  className?: string;
}

function DonutChart({ entries, total }: ChartEntry) {
  const size = 100;
  const strokeWidth = 12;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  
  let accumulatedOffset = 0;

  return (
    <div className="relative w-24 h-24 flex-shrink-0">
      <svg viewBox={`0 0 ${size} ${size}`} className="w-full h-full -rotate-90">
        {/* Background circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="var(--bg-tertiary)"
          strokeWidth={strokeWidth}
        />
        
        {/* Outcome segments */}
        {entries.map(([outcome, count], idx) => {
          const percentage = count / total;
          const dashLength = circumference * percentage;
          const dashOffset = circumference * accumulatedOffset;
          accumulatedOffset += percentage;
          
          return (
            <circle
              key={outcome}
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke={getOutcomeColor(outcome)}
              strokeWidth={strokeWidth}
              strokeDasharray={`${dashLength} ${circumference - dashLength}`}
              strokeDashoffset={-dashOffset}
              className="transition-all duration-300"
            />
          );
        })}
      </svg>
      
      {/* Center label */}
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-lg font-semibold text-[var(--text-primary)]">{total}</span>
        <span className="text-[10px] text-[var(--text-tertiary)]">total</span>
      </div>
    </div>
  );
}

function Legend({ entries, total }: ChartEntry) {
  return (
    <div className="flex flex-col gap-1.5">
      {entries.slice(0, 5).map(([outcome, count]) => (
        <div key={outcome} className="flex items-center gap-2 text-sm">
          <div 
            className="w-2.5 h-2.5 rounded-full flex-shrink-0"
            style={{ backgroundColor: getOutcomeColor(outcome) }}
          />
          <span className="text-[var(--text-secondary)] capitalize">{outcome}</span>
          <span className="text-[var(--text-tertiary)]">
            {count} ({((count / total) * 100).toFixed(0)}%)
          </span>
        </div>
      ))}
      {entries.length > 5 && (
        <span className="text-xs text-[var(--text-tertiary)] ml-4">
          +{entries.length - 5} more
        </span>
      )}
    </div>
  );
}

function BarChart({ entries, total, className }: ChartEntry) {
  const maxCount = Math.max(...entries.map(([, count]) => count));

  return (
    <div className={`space-y-2 ${className}`}>
      {entries.slice(0, 8).map(([outcome, count]) => (
        <div key={outcome} className="flex items-center gap-3">
          <span className="w-24 text-sm text-[var(--text-secondary)] capitalize truncate">
            {outcome}
          </span>
          <div className="flex-1 h-5 bg-[var(--bg-tertiary)] rounded overflow-hidden">
            <div
              className="h-full rounded transition-all duration-300"
              style={{
                width: `${(count / maxCount) * 100}%`,
                backgroundColor: getOutcomeColor(outcome),
              }}
            />
          </div>
          <span className="w-16 text-right text-sm text-[var(--text-tertiary)]">
            {count} ({((count / total) * 100).toFixed(0)}%)
          </span>
        </div>
      ))}
    </div>
  );
}
