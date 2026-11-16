import React from 'react';
import { AlertCircle, RotateCcw } from 'lucide-react';

export interface ErrorBannerProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  className?: string;
}

export const ErrorBanner: React.FC<ErrorBannerProps> = ({
  title = 'Failed to load data',
  message,
  onRetry,
  className = '',
}) => {
  return (
    <div
      className={`rounded-[10px] bg-[#f87171]/10 border border-[#f87171]/30 p-4 text-[#f3f4f8] flex items-start justify-between gap-3 ${className}`}
      role="alert"
    >
      <div className="flex items-start gap-3">
        <AlertCircle className="w-5 h-5 text-[#f87171] shrink-0 mt-0.5" />
        <div>
          <h5 className="text-[13px] font-semibold text-[#f87171]">{title}</h5>
          <p className="text-[12px] text-[#f3f4f8]/80 mt-0.5">{message}</p>
        </div>
      </div>

      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-[6px] text-[12px] font-medium bg-white/10 hover:bg-white/15 text-white transition-colors shrink-0"
        >
          <RotateCcw className="w-3.5 h-3.5" />
          <span>Retry</span>
        </button>
      )}
    </div>
  );
};

