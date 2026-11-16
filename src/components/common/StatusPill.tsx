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
      dot: 'bg-[#22c7d6]',
      text: 'text-[#22c7d6]',
      bg: 'bg-[#22c7d6]/10',
      border: 'border-[#22c7d6]/25',
      defaultLabel: 'Healthy',
    },
    ok: {
      dot: 'bg-[#34d399]',
      text: 'text-[#34d399]',
      bg: 'bg-[#34d399]/10',
      border: 'border-[#34d399]/25',
      defaultLabel: 'OK',
    },
    success: {
      dot: 'bg-[#34d399]',
      text: 'text-[#34d399]',
      bg: 'bg-[#34d399]/10',
      border: 'border-[#34d399]/25',
      defaultLabel: 'Success',
    },
    degraded: {
      dot: 'bg-[#f5b942]',
      text: 'text-[#f5b942]',
      bg: 'bg-[#f5b942]/10',
      border: 'border-[#f5b942]/25',
      defaultLabel: 'Degraded',
    },
    critical: {
      dot: 'bg-[#9b5cf6]',
      text: 'text-[#9b5cf6]',
      bg: 'bg-[#9b5cf6]/10',
      border: 'border-[#9b5cf6]/25',
      defaultLabel: 'Critical',
    },
    unhealthy: {
      dot: 'bg-[#9b5cf6]',
      text: 'text-[#9b5cf6]',
      bg: 'bg-[#9b5cf6]/10',
      border: 'border-[#9b5cf6]/25',
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
      dot: 'bg-[#f5b942]',
      text: 'text-[#f5b942]',
      bg: 'bg-[#f5b942]/10',
      border: 'border-[#f5b942]/25',
      defaultLabel: 'Open',
    },
    investigating: {
      dot: 'bg-[#60a5fa]',
      text: 'text-[#60a5fa]',
      bg: 'bg-[#60a5fa]/10',
      border: 'border-[#60a5fa]/25',
      defaultLabel: 'Investigating',
    },
    resolved: {
      dot: 'bg-[#34d399]',
      text: 'text-[#34d399]',
      bg: 'bg-[#34d399]/10',
      border: 'border-[#34d399]/25',
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

