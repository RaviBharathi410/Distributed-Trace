import React from 'react';

export type SeverityLevel = 'critical' | 'high' | 'medium' | 'low' | 'info';

export interface SeverityBadgeProps {
  severity: SeverityLevel | string;
  className?: string;
  size?: 'sm' | 'md';
}

export const SeverityBadge: React.FC<SeverityBadgeProps> = ({
  severity,
  className = '',
  size = 'md',
}) => {
  const normalized = severity.toLowerCase() as SeverityLevel;

  const styles: Record<
    SeverityLevel,
    { gradient: string; text: string; border: string; label: string }
  > = {
    critical: {
      gradient: 'bg-gradient-to-r from-white via-neutral-100 to-neutral-300',
      text: 'text-black font-extrabold',
      border: 'border border-white shadow-[0_0_8px_rgba(255,255,255,0.25)]',
      label: 'CRITICAL',
    },
    high: {
      gradient: 'bg-gradient-to-r from-neutral-200 via-neutral-300 to-neutral-400',
      text: 'text-black font-bold',
      border: 'border border-white/40',
      label: 'HIGH',
    },
    medium: {
      gradient: 'bg-gradient-to-r from-neutral-700 via-neutral-800 to-neutral-900',
      text: 'text-neutral-100 font-medium',
      border: 'border border-white/20',
      label: 'MEDIUM',
    },
    low: {
      gradient: 'bg-gradient-to-r from-neutral-900 to-black',
      text: 'text-neutral-400 font-normal',
      border: 'border border-white/10',
      label: 'LOW',
    },
    info: {
      gradient: 'bg-gradient-to-r from-neutral-900 to-black',
      text: 'text-neutral-400 font-normal',
      border: 'border border-white/10',
      label: 'INFO',
    },
  };

  const current = styles[normalized] || styles.info;
  const sizeClasses = size === 'sm' ? 'px-1.5 py-0.2 text-[9px]' : 'px-2 py-0.5 text-[10px]';

  return (
    <span
      className={`inline-flex items-center justify-center font-mono uppercase tracking-wider rounded-[6px] ${current.gradient} ${current.text} ${current.border} ${sizeClasses} select-none ${className}`}
    >
      {current.label}
    </span>
  );
};
