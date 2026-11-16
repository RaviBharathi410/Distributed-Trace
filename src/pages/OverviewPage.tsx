import React from 'react';
import { GlobalSearch } from '../components/shell/GlobalSearch';

export const OverviewPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      {/* Header with decorative wave line */}
      <div className="relative flex flex-col md:flex-row md:items-center justify-between gap-4 pb-2">
        <div className="z-10">
          <h1 className="text-[28px] font-semibold tracking-tight text-[#f3f4f8]">
            System Observability
          </h1>
          <p className="text-[13px] text-[#9aa0b5] mt-1">
            Real-time traces, service health, and performance across your distributed system.
          </p>
        </div>

        {/* Decorative subtle violet wave */}
        <div className="absolute right-0 top-0 w-80 h-16 pointer-events-none opacity-30 overflow-hidden" aria-hidden="true">
          <svg viewBox="0 0 320 60" fill="none" className="w-full h-full text-[#7c6cf0]">
            <path
              d="M0 30 C 50 10, 100 50, 150 25 C 200 0, 250 40, 320 20"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            />
            <path
              d="M20 40 C 70 20, 120 55, 170 35 C 220 15, 270 45, 320 30"
              stroke="currentColor"
              strokeWidth="1"
              strokeDasharray="4 4"
            />
          </svg>
        </div>
      </div>

      {/* Global Search Bar */}
      <GlobalSearch />

      {/* Main Content Placeholder for Phase 2 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {['Total Traces', 'p99 Latency', 'Active Services', 'Active Anomalies'].map((title) => (
          <div
            key={title}
            className="p-5 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]"
          >
            <div className="text-[12px] text-[#6b7188] uppercase tracking-wider">{title}</div>
            <div className="text-[28px] font-bold text-[#f3f4f8] mt-2">--</div>
          </div>
        ))}
      </div>
    </div>
  );
};

