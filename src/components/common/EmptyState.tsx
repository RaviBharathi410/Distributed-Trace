import React from 'react';
import { Inbox } from 'lucide-react';

export interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  title,
  description,
  icon,
  action,
  className = '',
}) => {
  return (
    <div
      className={`flex flex-col items-center justify-center text-center p-8 rounded-[14px] bg-[#10121f] border border-dashed border-[rgba(255,255,255,0.1)] ${className}`}
    >
      <div className="w-12 h-12 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-[#9aa0b5] mb-3">
        {icon || <Inbox className="w-6 h-6 text-[#6b7188]" />}
      </div>
      <h4 className="text-[14px] font-semibold text-[#f3f4f8]">{title}</h4>
      {description && (
        <p className="text-[12px] text-[#9aa0b5] max-w-sm mt-1 leading-relaxed">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
};

