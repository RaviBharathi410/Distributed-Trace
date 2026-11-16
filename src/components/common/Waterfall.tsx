import React from 'react';

export interface WaterfallSpan {
  id: string;
  service: string;
  operation: string;
  startMs: number;
  durationMs: number;
  depth?: number;
  isHot?: boolean;
  hasError?: boolean;
}

export interface WaterfallProps {
  spans: WaterfallSpan[];
  totalDurationMs?: number;
  className?: string;
}

export const Waterfall: React.FC<WaterfallProps> = ({
  spans,
  totalDurationMs,
  className = '',
}) => {
  const maxTime =
    totalDurationMs ||
    Math.max(
      ...spans.map((s) => s.startMs + s.durationMs),
      1
    );

  const timeGridSteps = [0, 0.25, 0.5, 0.75, 1];

  return (
    <div className={`w-full flex flex-col gap-2 ${className}`}>
      {/* Time axis header */}
      <div className="flex items-center text-[11px] font-mono text-[#6b7188] pb-1 border-b border-[rgba(255,255,255,0.06)] pl-48 sm:pl-64 pr-2">
        <div className="w-full flex justify-between">
          {timeGridSteps.map((step) => (
            <span key={step}>{Math.round(step * maxTime)}ms</span>
          ))}
        </div>
      </div>

      {/* Spans list */}
      <div className="flex flex-col divide-y divide-[rgba(255,255,255,0.03)]">
        {spans.map((span) => {
          const depth = span.depth || 0;
          const leftPercent = Math.min(100, Math.max(0, (span.startMs / maxTime) * 100));
          const widthPercent = Math.min(
            100 - leftPercent,
            Math.max(1, (span.durationMs / maxTime) * 100)
          );

          return (
            <div
              key={span.id}
              className="py-2.5 flex items-center hover:bg-white/[0.02] transition-colors group"
            >
              {/* Span metadata (left column) */}
              <div
                className="w-48 sm:w-64 shrink-0 pr-3 flex items-center truncate"
                style={{ paddingLeft: `${depth * 14}px` }}
              >
                <div className="flex items-center gap-1.5 truncate">
                  <span
                    className={`w-2 h-2 rounded-full shrink-0 ${
                      span.hasError
                        ? 'bg-[#f87171]'
                        : span.isHot
                        ? 'bg-[#9b6bf2]'
                        : 'bg-[#4d8dff]'
                    }`}
                  />
                  <span
                    className={`text-[12px] font-medium truncate ${
                      span.isHot ? 'text-[#b08cff]' : 'text-[#f3f4f8]'
                    }`}
                  >
                    {span.service}
                  </span>
                  <span className="text-[11px] text-[#6b7188] truncate font-mono">
                    /{span.operation}
                  </span>
                </div>
              </div>

              {/* Timeline bar (right column) */}
              <div className="flex-1 relative h-6 flex items-center">
                {/* Subtle vertical grid lines */}
                <div className="absolute inset-0 flex justify-between pointer-events-none opacity-10">
                  {timeGridSteps.map((step) => (
                    <div key={step} className="w-px h-full bg-white" />
                  ))}
                </div>

                {/* Progress bar */}
                <div
                  className={`h-4 rounded-[4px] relative group-hover:brightness-110 transition-all shadow-sm ${
                    span.isHot
                      ? 'bg-gradient-to-r from-[#9b6bf2] to-[#b08cff]'
                      : 'bg-gradient-to-r from-[#3b6cf0] to-[#5b8cff]'
                  }`}
                  style={{
                    left: `${leftPercent}%`,
                    width: `${widthPercent}%`,
                  }}
                  title={`${span.service}/${span.operation}: ${span.durationMs}ms`}
                >
                  <span className="absolute left-full ml-2 top-1/2 -translate-y-1/2 text-[10px] font-mono text-[#9aa0b5] whitespace-nowrap">
                    {span.durationMs}ms
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

