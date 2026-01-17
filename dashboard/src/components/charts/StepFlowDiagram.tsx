'use client';

interface StepFlowDiagramProps {
  steps: StepData[];
  className?: string;
}

export interface StepData {
  type: string;
  inputCount: number;
  outputCount: number;
  durationMs: number;
}

/**
 * Visual pipeline flow diagram showing steps with reduction visualization
 * Steps are displayed as a horizontal flow with widths proportional to item counts
 */
export function StepFlowDiagram({ steps, className = '' }: StepFlowDiagramProps) {
  if (steps.length === 0) {
    return (
      <div className={`text-sm text-[var(--text-tertiary)] p-4 ${className}`}>
        No step data available
      </div>
    );
  }

  const maxCount = Math.max(...steps.flatMap(s => [s.inputCount, s.outputCount]));

  return (
    <div className={`flex items-center gap-1 overflow-x-auto py-4 ${className}`}>
      {steps.map((step, idx) => {
        const inputWidth = Math.max(20, (step.inputCount / maxCount) * 100);
        const outputWidth = Math.max(20, (step.outputCount / maxCount) * 100);
        const reduction = step.inputCount > 0 
          ? 1 - (step.outputCount / step.inputCount) 
          : 0;
        
        const isHighReduction = reduction >= 0.5;
        const isBottleneck = step.durationMs > 1000; // > 1s is slow

        return (
          <div key={step.type} className="flex items-center">
            {/* Step block */}
            <div className="flex flex-col items-center group">
              {/* Input bar */}
              <div 
                className="h-2 rounded-t bg-[var(--accent)]/40 transition-all"
                style={{ width: `${inputWidth}px` }}
              />
              
              {/* Step label */}
              <div className={`
                px-3 py-2 rounded border text-center min-w-[80px]
                transition-all cursor-default
                ${isBottleneck ? 'border-[var(--warning)] bg-[var(--warning)]/5' : 'border-[var(--border-primary)] bg-[var(--bg-secondary)]'}
                ${isHighReduction ? 'ring-1 ring-[var(--error)]/30' : ''}
                group-hover:border-[var(--border-secondary)]
              `}>
                <span className="text-xs font-medium text-[var(--text-primary)] block truncate max-w-[120px]">
                  {step.type}
                </span>
              </div>
              
              {/* Output bar */}
              <div 
                className={`h-2 rounded-b transition-all ${
                  isHighReduction ? 'bg-[var(--error)]/60' : 'bg-[var(--accent)]/60'
                }`}
                style={{ width: `${outputWidth}px` }}
              />
              
              {/* Stats */}
              <div className="mt-1 text-[10px] text-[var(--text-tertiary)] text-center">
                {step.inputCount} → {step.outputCount}
                {reduction > 0 && (
                  <span className={isHighReduction ? 'text-[var(--error)]' : ''}>
                    {' '}({(reduction * 100).toFixed(0)}%)
                  </span>
                )}
              </div>
            </div>
            
            {/* Connector arrow */}
            {idx < steps.length - 1 && (
              <div className="flex items-center px-1 text-[var(--text-tertiary)]">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
