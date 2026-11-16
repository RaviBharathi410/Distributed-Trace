import React from 'react';

export interface HorizontalBarItem {
  id: string;
  label: string;
  value: number;
  formattedValue?: string;
  isHot?: boolean;
}

export interface HorizontalBarsProps {
  items: HorizontalBarItem[];
  valuePrefix?: string;
  valueSuffix?: string;
  className?: string;
}

export const HorizontalBars: React.FC<HorizontalBarsProps> = ({
  items,
  valuePrefix = '',
  valueSuffix = '',
  className = '',
}) => {
  if (!items || items.length === 0) {
    return <div className="text-center py-6 text-[#6b7188] text-[12px]">No data available</div>;
  }

  const maxValue = Math.max(...items.map((it) => it.value), 1);
  const totalValue = items.reduce((sum, it) => sum + it.value, 0);

  return (
    <div className={`flex flex-col gap-3.5 w-full ${className}`}>
      {items.map((item) => {
        const widthPct = Math.max(2, (item.value / maxValue) * 100);
        const sharePct = totalValue > 0 ? ((item.value / totalValue) * 100).toFixed(1) : '0';

        return (
          <div key={item.id} className="flex flex-col gap-1.5 group">
            <div className="flex items-center justify-between text-[12px]">
              <span className={`font-medium truncate ${item.isHot ? 'text-[#b08cff]' : 'text-[#f3f4f8]'}`}>
                {item.label}
              </span>
              <div className="flex items-center gap-2 font-mono text-[11px]">
                <span className="text-[#f3f4f8] font-semibold">
                  {item.formattedValue || `${valuePrefix}${item.value.toLocaleString()}${valueSuffix}`}
                </span>
                <span className="text-[#6b7188]">({sharePct}%)</span>
              </div>
            </div>

            <div className="w-full h-2 rounded-full bg-white/5 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-300 group-hover:brightness-110 ${
                  item.isHot
                    ? 'bg-gradient-to-r from-[#9b6bf2] to-[#b08cff]'
                    : 'bg-gradient-to-r from-[#3b6cf0] to-[#5b8cff]'
                }`}
                style={{ width: `${widthPct}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
};

