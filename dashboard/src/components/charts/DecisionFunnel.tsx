import React from 'react';
import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  Cell
} from 'recharts';

interface FunnelStep {
  name: string;
  value: number;
  dropOff?: number;
}

interface DecisionFunnelProps {
  data: FunnelStep[];
  className?: string;
}

export default function DecisionFunnel({ data, className = '' }: DecisionFunnelProps) {
  // Sort data by funnel order (assuming input data is already ordered by step flow)
  // We can calculate conversion rates
  
  const formattedData = data.map((item, index) => {
    const prevValue = index > 0 ? data[index - 1].value : item.value;
    const conversion = prevValue > 0 ? (item.value / prevValue) * 100 : 0;
    
    return {
      ...item,
      conversion: index === 0 ? 100 : conversion,
      fill: index === data.length - 1 ? 'var(--success)' : 'var(--accent)',
    };
  });

  return (
    <div className={`w-full h-[300px] ${className}`}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={formattedData}
          layout="vertical"
          margin={{ top: 20, right: 30, left: 40, bottom: 5 }}
        >
          <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="var(--border-primary)" opacity={0.3} />
          <XAxis type="number" hide />
          <YAxis 
            dataKey="name" 
            type="category" 
            tick={{ fill: 'var(--text-secondary)', fontSize: 12 }} 
            width={120}
          />
          <Tooltip 
            cursor={{ fill: 'var(--bg-tertiary)', opacity: 0.4 }}
            content={({ active, payload }) => {
              if (active && payload && payload.length) {
                const d = payload[0].payload;
                return (
                  <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] p-3 rounded shadow-xl">
                    <p className="font-medium text-[var(--text-primary)] mb-1">{d.name}</p>
                    <div className="space-y-1 text-sm">
                      <p className="flex justify-between gap-4">
                        <span className="text-[var(--text-tertiary)]">Items:</span>
                        <span className="font-mono text-[var(--text-primary)]">{d.value}</span>
                      </p>
                      {d.conversion < 100 && (
                        <p className="flex justify-between gap-4">
                          <span className="text-[var(--text-tertiary)]">Conversion:</span>
                          <span className={d.conversion > 80 ? 'text-[var(--success)]' : d.conversion > 50 ? 'text-[var(--warning)]' : 'text-[var(--error)]'}>
                            {d.conversion.toFixed(1)}%
                          </span>
                        </p>
                      )}
                    </div>
                  </div>
                );
              }
              return null;
            }}
          />
          <Bar dataKey="value" radius={[0, 4, 4, 0]} barSize={32}>
            {formattedData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.value === 0 ? 'var(--text-tertiary)' : entry.fill} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
