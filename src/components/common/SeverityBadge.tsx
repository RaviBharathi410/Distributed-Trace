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

  const styles: Record<SeverityLevel, { bg: string; text: string; label: string }> = {
    critical: {
      bg: 'bg-[#9b5cf6]',
      text: 'text-white',
      label: 'CRITICAL',
    },
    high: {
      bg: 'bg-[#3b82f6]',
      text: 'text-white',
      label: 'HIGH',
    },
    medium: {
      bg: 'bg-[#818cf8]',
      text: 'text-white',
      label: 'MEDIUM',
    },
    low: {
      bg: 'bg-[#64748b]',
      text: 'text-white',
      label: 'LOW',
    },
    info: {
      bg: 'bg-[#64748b]',
      text: 'text-white',
      label: 'INFO',
    },
  };

  const current = styles[normalized] || styles.info;
  const sizeClasses = size === 'sm' ? 'px-1.5 py-0.2 text-[9px]' : 'px-2 py-0.5 text-[10px]';

  return (
    <span
      className={`inline-flex items-center justify-center font-bold uppercase tracking-wider rounded-[6px] ${current.bg} ${current.text} ${sizeClasses} shadow-sm ${className}`}
    >
      {current.label}
    </span>
  );
};

