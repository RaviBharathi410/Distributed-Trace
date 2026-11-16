import React from 'react';

export interface BarDatum {
  label: string;
  value: number;
  sublabel?: string;
  isHot?: boolean;
}

export interface BarChartProps {
  data: BarDatum[];
  height?: number;
  valuePrefix?: string;
  valueSuffix?: string;
  className?: string;
}

export const BarChart: React.FC<BarChartProps> = ({
  data,
  height = 180,
  valuePrefix = '',
  valueSuffix = '',
  className = '',
}) => {
  if (!data || data.length === 0) {
    return <div className="h-40 flex items-center justify-center text-[#6b7188] text-[12px]">No data</div>;
  }

  const maxValue = Math.max(...data.map((d) => d.value), 1);

  return (
    <div className={`w-full flex flex-col gap-2 ${className}`}>
      <div className="flex items-end gap-2 sm:gap-3 justify-between pt-4 pb-2" style={{ height }}>
        {data.map((item, idx) => {
          const heightPct = Math.max(4, (item.value / maxValue) * 100);

          return (
            <div
              key={`bar-${idx}`}
              className="flex-1 flex flex-col items-center h-full justify-end group relative"
            >
              {/* Tooltip on hover */}
              <div className="absolute bottom-full mb-2 hidden group-hover:flex flex-col items-center pointer-events-none z-20">
                <div className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] text-[#f3f4f8] text-[11px] font-mono px-2 py-1 rounded shadow-lg whitespace-nowrap">
                  {valuePrefix}
                  {item.value.toLocaleString()}
                  {valueSuffix}
                </div>
                <div className="w-1.5 h-1.5 bg-[#0a0b14] border-r border-b border-[rgba(255,255,255,0.12)] rotate-45 -mt-1" />
              </div>

              {/* Bar */}
              <div
                className={`w-full max-w-[36px] rounded-t-[6px] transition-all duration-200 group-hover:brightness-110 shadow-sm ${
                  item.isHot
                    ? 'bg-gradient-to-t from-[#9b6bf2] to-[#b08cff]'
                    : 'bg-gradient-to-t from-[#3b6cf0] to-[#5b8cff]'
                }`}
                style={{ height: `${heightPct}%` }}
              />
            </div>
          );
        })}
      </div>

      {/* X-axis labels */}
      <div className="flex items-center justify-between gap-1 border-t border-[rgba(255,255,255,0.06)] pt-2">
        {data.map((item, idx) => (
          <div
            key={`label-${idx}`}
            className="flex-1 text-center text-[10px] font-mono text-[#6b7188] truncate"
            title={item.label}
          >
            {item.label}
          </div>
        ))}
      </div>
    </div>
  );
};

