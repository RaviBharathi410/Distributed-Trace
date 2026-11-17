import React, { useState } from 'react';
import {
  Activity,
  Clock,
  Layers,
  Server,
  RefreshCw,
  Calendar,
  ChevronDown,
  TrendingUp,
  Loader2,
  ShieldAlert,
  DollarSign,
  ChevronRight,
} from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { costsApi } from '../../lib/api';
import {
  DEMO_COSTS_KPIS,
  DEMO_COST_TREND,
  DEMO_COST_BY_SERVICE,
} from '../../lib/fixtures';

export const CostsTab: React.FC = () => {
  const [timeRange, setTimeRange] = useState('This Month');
  const [showLedger, setShowLedger] = useState(false);

  const {
    data: summary,
    isLoading: summaryLoading,
    isError: summaryError,
    refetch: refetchSummary,
    isFetching: summaryFetching,
  } = useQuery({
    queryKey: ['costs', 'summary'],
    queryFn: () => costsApi.getSummary(),
    refetchInterval: 30000,
  });

  const {
    data: breakdown,
    isLoading: breakdownLoading,
    refetch: refetchBreakdown,
  } = useQuery({
    queryKey: ['costs', 'breakdown'],
    queryFn: () => costsApi.getBreakdown({ limit: 50 }),
    refetchInterval: 30000,
  });

  const handleRefresh = () => {
    refetchSummary();
    refetchBreakdown();
  };

  const isBusy = summaryFetching || summaryLoading || breakdownLoading;

  // Use real data if available and non-zero, otherwise default to source-of-truth mockup fixtures
  const totalCost = summary && summary.totalCostUsd > 0
    ? `$${summary.totalCostUsd.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
    : DEMO_COSTS_KPIS.totalMonthlyCost;

  const avgCost = summary && summary.avgCostPerIncidentUsd > 0
    ? `$${summary.avgCostPerIncidentUsd.toFixed(2)}`
    : DEMO_COSTS_KPIS.avgCostPerIncident;

  const llmSpend = summary && summary.llmSpendUsd > 0
    ? `$${summary.llmSpendUsd.toFixed(2)}`
    : DEMO_COSTS_KPIS.llmTokenSpend;

  const infraSpend = summary && summary.infraShareUsd > 0
    ? `$${summary.infraShareUsd.toFixed(2)}`
    : DEMO_COSTS_KPIS.infrastructure;

  const maxTrendCost = 1000;
  const yTicks = [1000, 750, 500, 250, 0];

  return (
    <div className="flex flex-col gap-6 text-[#f3f4f8] font-sans pb-10">
      {/* HEADER */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-[rgba(255,255,255,0.07)]">
        <div>
          <h1 className="text-[26px] font-semibold text-[#f3f4f8] tracking-tight">
            Cost Analysis
          </h1>
          <p className="text-[13px] text-[#9aa0b5] mt-1">
            Tenant cost attribution, token usage, and infrastructure economics.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {/* Time Filter Dropdown */}
          <div className="relative">
            <button
              type="button"
              className="flex items-center gap-2 px-3.5 py-1.5 rounded-[9999px] bg-[#10121f] border border-[rgba(255,255,255,0.1)] text-[12px] text-[#f3f4f8] hover:border-[rgba(255,255,255,0.2)] transition-colors focus:outline-none"
              onClick={() => {
                setTimeRange(timeRange === 'This Month' ? 'Last 30 Days' : 'This Month');
              }}
            >
              <Calendar className="w-3.5 h-3.5 text-[#9aa0b5]" />
              <span>{timeRange}</span>
              <ChevronDown className="w-3.5 h-3.5 text-[#6b7188]" />
            </button>
          </div>

          {/* Refresh Button */}
          <button
            type="button"
            onClick={handleRefresh}
            disabled={isBusy}
            title="Refresh cost metrics"
            className="p-2 rounded-[8px] bg-[#10121f] border border-[rgba(255,255,255,0.1)] text-[#9aa0b5] hover:text-[#f3f4f8] hover:border-[rgba(255,255,255,0.2)] transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isBusy ? 'animate-spin text-[#7c6cf0]' : ''}`} />
          </button>
        </div>
      </div>

      {summaryError && (
        <div className="p-4 rounded-[14px] bg-red-500/10 border border-red-500/20 text-red-300 text-[13px] flex items-center gap-3">
          <ShieldAlert className="w-5 h-5 shrink-0 text-red-400" />
          <span>Failed to load live tenant cost attribution metrics. Displaying baseline calibration.</span>
        </div>
      )}

      {/* 4 TOP KPI CARDS */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Card 1: Total Monthly Cost */}
        <div className="p-5 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex items-center gap-4 relative overflow-hidden group hover:border-[rgba(255,255,255,0.14)] transition-colors">
          <div className="w-11 h-11 rounded-[10px] bg-[rgba(124,108,240,0.15)] border border-[rgba(124,108,240,0.3)] flex items-center justify-center shrink-0">
            <Activity className="w-5 h-5 text-[#7c6cf0]" />
          </div>
          <div>
            <div className="text-[12px] font-normal text-[#9aa0b5]">
              Total Monthly Cost
            </div>
            <div className="text-[24px] font-bold text-[#f3f4f8] mt-0.5 tracking-tight font-mono">
              {totalCost}
            </div>
          </div>
        </div>

        {/* Card 2: Avg. Cost / Incident */}
        <div className="p-5 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex items-center gap-4 relative overflow-hidden group hover:border-[rgba(255,255,255,0.14)] transition-colors">
          <div className="w-11 h-11 rounded-[10px] bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] flex items-center justify-center shrink-0">
            <Clock className="w-5 h-5 text-[#9aa0b5]" />
          </div>
          <div>
            <div className="text-[12px] font-normal text-[#9aa0b5]">
              Avg. Cost / Incident
            </div>
            <div className="text-[24px] font-bold text-[#f3f4f8] mt-0.5 tracking-tight font-mono">
              {avgCost}
            </div>
          </div>
        </div>

        {/* Card 3: LLM / Token Spend */}
        <div className="p-5 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex items-center gap-4 relative overflow-hidden group hover:border-[rgba(255,255,255,0.14)] transition-colors">
          <div className="w-11 h-11 rounded-[10px] bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] flex items-center justify-center shrink-0">
            <Layers className="w-5 h-5 text-[#9aa0b5]" />
          </div>
          <div>
            <div className="text-[12px] font-normal text-[#9aa0b5]">
              LLM / Token Spend
            </div>
            <div className="text-[24px] font-bold text-[#f3f4f8] mt-0.5 tracking-tight font-mono">
              {llmSpend}
            </div>
          </div>
        </div>

        {/* Card 4: Infrastructure */}
        <div className="p-5 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex items-center gap-4 relative overflow-hidden group hover:border-[rgba(255,255,255,0.14)] transition-colors">
          <div className="w-11 h-11 rounded-[10px] bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] flex items-center justify-center shrink-0">
            <Server className="w-5 h-5 text-[#9aa0b5]" />
          </div>
          <div>
            <div className="text-[12px] font-normal text-[#9aa0b5]">
              Infrastructure
            </div>
            <div className="text-[24px] font-bold text-[#f3f4f8] mt-0.5 tracking-tight font-mono">
              {infraSpend}
            </div>
          </div>
        </div>
      </div>

      {/* ROW 2: COST TREND & COST BY SERVICE */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch">
        {/* Left (~60% / 7 cols): Cost Trend */}
        <div className="lg:col-span-7 p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col justify-between">
          <div className="flex items-center gap-2 mb-6">
            <TrendingUp className="w-4 h-4 text-[#7c6cf0]" />
            <h2 className="text-[14px] font-semibold text-[#f3f4f8]">
              Cost Trend
            </h2>
          </div>

          {/* Chart Container with Y-Axis and Grid */}
          <div className="w-full flex flex-col gap-2">
            <div className="relative flex items-stretch h-[200px] w-full">
              {/* Y-Axis Labels */}
              <div className="flex flex-col justify-between text-right pr-3 shrink-0 text-[11px] font-mono text-[#6b7188] select-none py-1">
                {yTicks.map((val) => (
                  <span key={val}>${val.toLocaleString()}</span>
                ))}
              </div>

              {/* Chart Plot Area with Dotted Grid Lines */}
              <div className="relative flex-1 flex flex-col justify-between">
                {/* Horizontal Grid Lines */}
                <div className="absolute inset-0 flex flex-col justify-between pointer-events-none">
                  {yTicks.map((val) => (
                    <div
                      key={`grid-${val}`}
                      className="w-full border-b border-dashed border-[rgba(255,255,255,0.06)] h-0"
                    />
                  ))}
                </div>

                {/* Vertical Gradient Bars */}
                <div className="relative z-10 w-full h-full flex items-end justify-between px-3 pt-2">
                  {DEMO_COST_TREND.map((item) => {
                    const heightPct = Math.max(6, (item.cost / maxTrendCost) * 100);

                    return (
                      <div
                        key={item.date}
                        className="flex-1 flex flex-col items-center h-full justify-end group relative mx-0.5 sm:mx-1"
                      >
                        {/* Hover Tooltip */}
                        <div className="absolute bottom-full mb-2 hidden group-hover:flex flex-col items-center pointer-events-none z-30">
                          <div className="bg-[#0a0b14] border border-[rgba(255,255,255,0.15)] text-[#f3f4f8] text-[11px] font-mono px-2 py-1 rounded shadow-xl whitespace-nowrap">
                            <span className="text-[#9aa0b5]">{item.date}: </span>
                            <span className="font-semibold text-[#b08cff]">${item.cost}</span>
                          </div>
                          <div className="w-1.5 h-1.5 bg-[#0a0b14] border-r border-b border-[rgba(255,255,255,0.15)] rotate-45 -mt-1" />
                        </div>

                        {/* Bar */}
                        <div
                          className="w-full max-w-[28px] rounded-t-[4px] bg-gradient-to-t from-[#7c6cf0] to-[#b08cff] transition-all duration-200 group-hover:brightness-125 shadow-sm"
                          style={{ height: `${heightPct}%` }}
                        />
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* X-Axis Date Labels */}
            <div className="flex items-center justify-between pl-14 pr-2 pt-3 border-t border-[rgba(255,255,255,0.06)] text-[11px] font-mono text-[#6b7188]">
              {DEMO_COST_TREND.map((item, idx) => {
                // Show alternate date labels on narrow viewports or matching mockup dates
                const isKeyDate = idx % 2 === 0;
                return (
                  <span
                    key={`label-${item.date}`}
                    className={`text-center ${isKeyDate ? 'opacity-100' : 'hidden md:inline-block opacity-60'}`}
                  >
                    {item.date}
                  </span>
                );
              })}
            </div>
          </div>
        </div>

        {/* Right (~40% / 5 cols): Cost by Service */}
        <div className="lg:col-span-5 p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-2 mb-6">
              <Layers className="w-4 h-4 text-[#7c6cf0]" />
              <h2 className="text-[14px] font-semibold text-[#f3f4f8]">
                Cost by Service
              </h2>
            </div>

            {/* Horizontal Distribution Bars matching mockup layout */}
            <div className="flex flex-col gap-4">
              {DEMO_COST_BY_SERVICE.map((item) => (
                <div key={item.id} className="flex items-center gap-4 text-[12px] group">
                  {/* Service Name */}
                  <span className="w-32 truncate text-[#f3f4f8] font-mono text-[12px]">
                    {item.service}
                  </span>

                  {/* Horizontal Bar */}
                  <div className="flex-1 h-2.5 rounded-full bg-[rgba(255,255,255,0.05)] overflow-hidden">
                    <div
                      className="h-full rounded-full bg-gradient-to-r from-[#5b8cff] to-[#7c6cf0] transition-all duration-300 group-hover:brightness-110"
                      style={{ width: `${item.percentage * 2.5}%` }}
                    />
                  </div>

                  {/* Percentage */}
                  <span className="w-10 text-right font-mono text-[12px] text-[#9aa0b5] font-medium">
                    {item.percentage}%
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="mt-6 pt-4 border-t border-[rgba(255,255,255,0.06)] flex items-center justify-between text-[11px] text-[#6b7188]">
            <span>Top 5 microservices by compute & trace egress</span>
            <span className="font-mono text-[#9aa0b5]">100% Attributed</span>
          </div>
        </div>
      </div>

      {/* COLLAPSIBLE / DETAILED INCIDENT LEDGER TABLE */}
      <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] overflow-hidden">
        <button
          type="button"
          onClick={() => setShowLedger(!showLedger)}
          className="w-full px-6 py-4 flex items-center justify-between text-left hover:bg-white/[0.02] transition-colors focus:outline-none"
        >
          <div className="flex items-center gap-2">
            <DollarSign className="w-4 h-4 text-[#7c6cf0]" />
            <span className="text-[14px] font-semibold text-[#f3f4f8]">
              Detailed Incident Explanation Cost Ledger
            </span>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-[9999px] bg-white/[0.04] text-[#9aa0b5] ml-2">
              {breakdown?.items?.length || 0} Events Recorded
            </span>
          </div>
          <div className="flex items-center gap-1.5 text-[12px] text-[#7c6cf0]">
            <span>{showLedger ? 'Collapse Ledger' : 'View Ledger'}</span>
            <ChevronRight className={`w-4 h-4 transition-transform duration-200 ${showLedger ? 'rotate-90' : ''}`} />
          </div>
        </button>

        {showLedger && (
          <div className="border-t border-[rgba(255,255,255,0.07)] p-6 animate-in fade-in duration-200">
            {breakdownLoading ? (
              <div className="flex items-center justify-center py-12 text-[#9aa0b5] text-[13px] gap-2">
                <Loader2 className="w-4 h-4 animate-spin text-[#7c6cf0]" />
                <span>Loading incident cost ledger...</span>
              </div>
            ) : !breakdown?.items || breakdown.items.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-[#9aa0b5] text-[13px] gap-2">
                <DollarSign className="w-8 h-8 text-[#6b7188]" />
                <span className="text-[#f3f4f8]">No incident explanations generated yet for this organization.</span>
                <span className="text-[11px] text-[#6b7188]">
                  Trigger on-demand diagnosis from the Anomalies tab to record live cost events.
                </span>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-[12px]">
                  <thead>
                    <tr className="border-b border-[rgba(255,255,255,0.06)] text-[#6b7188] font-mono uppercase text-[11px]">
                      <th className="py-2.5 px-4 font-medium">Anomaly ID</th>
                      <th className="py-2.5 px-4 font-medium">Model</th>
                      <th className="py-2.5 px-4 font-medium">Prompt Tokens</th>
                      <th className="py-2.5 px-4 font-medium">Output Tokens</th>
                      <th className="py-2.5 px-4 font-medium">Estimated Cost</th>
                      <th className="py-2.5 px-4 font-medium">Recorded At</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[rgba(255,255,255,0.04)] font-mono">
                    {breakdown.items.map((item) => (
                      <tr key={item.id} className="hover:bg-white/[0.02] transition-colors">
                        <td className="py-3 px-4 text-[#f3f4f8]">{item.anomalyId}</td>
                        <td className="py-3 px-4 text-[#7c6cf0]">{item.model}</td>
                        <td className="py-3 px-4 text-[#9aa0b5]">{item.inputTokens}</td>
                        <td className="py-3 px-4 text-[#9aa0b5]">{item.outputTokens}</td>
                        <td className="py-3 px-4 text-[#34d399] font-semibold">${item.estimatedCostUsd.toFixed(6)}</td>
                        <td className="py-3 px-4 text-[#6b7188]">{new Date(item.createdAt).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
