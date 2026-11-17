import React from 'react';
import { ArrowUpRight, ArrowDownRight, Minus } from 'lucide-react';

export interface StatCardProps {
  label: string;
  value: string | number;
  icon?: React.ReactNode;
  iconColor?: string;
  iconBg?: string;
  delta?: number;
  deltaLabel?: string;
  invertDelta?: boolean; // if true, negative delta is green (e.g. lower latency is good)
  subtext?: string;
  className?: string;
}

export const StatCard: React.FC<StatCardProps> = ({
  label,
  value,
  icon,
  iconColor = '#7c6cf0',
  iconBg = 'rgba(124, 108, 240, 0.12)',
  delta,
  deltaLabel,
  invertDelta = false,
  subtext,
  className = '',
}) => {
  let isPositive = false;
  let isNeutral = true;

  if (delta !== undefined && delta !== null) {
    if (delta > 0) {
      isPositive = !invertDelta;
      isNeutral = false;
    } else if (delta < 0) {
      isPositive = invertDelta;
      isNeutral = false;
    }
  }

  return (
    <div
      className={`rounded-[14px] bg-[#0a0a0d] border border-white/[0.08] p-5 flex items-start gap-4 transition-all duration-150 hover:border-white/[0.14] ${className}`}
    >
      {icon && (
        <div
          className="w-10 h-10 rounded-[10px] flex items-center justify-center shrink-0 border border-white/[0.05]"
          style={{ backgroundColor: iconBg, color: iconColor }}
        >
          {icon}
        </div>
      )}

      <div className="flex-1 min-w-0">
        <div className="text-[12px] font-medium text-[#9aa0b5] tracking-wide truncate">
          {label}
        </div>

        <div className="flex items-baseline gap-2.5 mt-1.5 flex-wrap">
          <div className="text-[28px] font-bold text-[#f3f4f8] tracking-tight leading-none">
            {value}
          </div>

          {delta !== undefined && delta !== null && (
            <div
              className={`inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-[6px] text-[11px] font-medium leading-none ${
                isNeutral
                  ? 'bg-white/5 text-[#9aa0b5]'
                  : isPositive
                  ? 'bg-[#3b82f6]/15 text-[#60a5fa] border border-[#3b82f6]/30'
                  : 'bg-white/10 text-white/80 border border-white/20'
              }`}
            >
              {delta > 0 ? (
                <ArrowUpRight className="w-3 h-3" />
              ) : delta < 0 ? (
                <ArrowDownRight className="w-3 h-3" />
              ) : (
                <Minus className="w-3 h-3" />
              )}
              <span>
                {delta > 0 ? '+' : ''}
                {delta.toFixed(1)}%
              </span>
            </div>
          )}
        </div>

        {(deltaLabel || subtext) && (
          <div className="text-[11px] text-[#6b7188] mt-1.5 truncate">
            {deltaLabel || subtext}
          </div>
        )}
      </div>
    </div>
  );
};

