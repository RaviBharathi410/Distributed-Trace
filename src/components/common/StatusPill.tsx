import React from 'react';

export type StatusType =
  | 'healthy'
  | 'degraded'
  | 'critical'
  | 'unhealthy'
  | 'success'
  | 'ok'
  | 'error'
  | 'open'
  | 'investigating'
  | 'resolved';

export interface StatusPillProps {
  status: StatusType | string;
  label?: string;
  showDot?: boolean;
  className?: string;
}

export const StatusPill: React.FC<StatusPillProps> = ({
  status,
  label,
  showDot = true,
  className = '',
}) => {
  const norm = status.toLowerCase();

  const configs: Record<
    string,
    { dot: string; text: string; bg: string; border: string; defaultLabel: string }
  > = {
    healthy: {
      dot: 'bg-[#60a5fa]',
      text: 'text-[#60a5fa]',
      bg: 'bg-[#3b82f6]/10',
      border: 'border-[#3b82f6]/30',
      defaultLabel: 'Healthy',
    },
    ok: {
      dot: 'bg-[#60a5fa]',
      text: 'text-[#60a5fa]',
      bg: 'bg-[#3b82f6]/10',
      border: 'border-[#3b82f6]/30',
      defaultLabel: 'OK',
    },
    success: {
      dot: 'bg-[#60a5fa]',
      text: 'text-[#60a5fa]',
      bg: 'bg-[#3b82f6]/10',
      border: 'border-[#3b82f6]/30',
      defaultLabel: 'Success',
    },
    degraded: {
      dot: 'bg-[#818cf8]',
      text: 'text-[#818cf8]',
      bg: 'bg-[#818cf8]/10',
      border: 'border-[#818cf8]/30',
      defaultLabel: 'Degraded',
    },
    critical: {
      dot: 'bg-[#9b5cf6]',
      text: 'text-[#9b5cf6]',
      bg: 'bg-[#9b5cf6]/10',
      border: 'border-[#9b5cf6]/30',
      defaultLabel: 'Critical',
    },
    unhealthy: {
      dot: 'bg-[#9b5cf6]',
      text: 'text-[#9b5cf6]',
      bg: 'bg-[#9b5cf6]/10',
      border: 'border-[#9b5cf6]/30',
      defaultLabel: 'Unhealthy',
    },
    error: {
      dot: 'bg-[#f87171]',
      text: 'text-[#f87171]',
      bg: 'bg-[#f87171]/10',
      border: 'border-[#f87171]/25',
      defaultLabel: 'Error',
    },
    open: {
      dot: 'bg-[#60a5fa]',
      text: 'text-[#60a5fa]',
      bg: 'bg-[#3b82f6]/10',
      border: 'border-[#3b82f6]/30',
      defaultLabel: 'Open',
    },
    investigating: {
      dot: 'bg-[#818cf8]',
      text: 'text-[#818cf8]',
      bg: 'bg-[#818cf8]/10',
      border: 'border-[#818cf8]/30',
      defaultLabel: 'Investigating',
    },
    resolved: {
      dot: 'bg-[#60a5fa]',
      text: 'text-[#60a5fa]',
      bg: 'bg-[#3b82f6]/10',
      border: 'border-[#3b82f6]/30',
      defaultLabel: 'Resolved',
    },
  };

  const cfg = configs[norm] || {
    dot: 'bg-[#9aa0b5]',
    text: 'text-[#9aa0b5]',
    bg: 'bg-white/5',
    border: 'border-white/10',
    defaultLabel: status,
  };

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium border ${cfg.bg} ${cfg.border} ${cfg.text} ${className}`}
    >
      {showDot && <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />}
      <span>{label || cfg.defaultLabel}</span>
    </span>
  );
};

