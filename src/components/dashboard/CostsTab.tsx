import React from 'react';
import { DollarSign, ShieldAlert, Layers, RefreshCw, Loader2, CheckCircle2, Clock, Sparkles } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { costsApi } from '../../lib/api';

export const CostsTab: React.FC = () => {
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

  return (
    <div className="flex flex-col h-full gap-6 text-white font-dm-sans overflow-y-auto pr-2">
      {/* HEADER */}
      <div className="flex items-center justify-between border-b border-white/[0.08] pb-5 shrink-0">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-xl font-bold tracking-tight text-white">Tenant Cost Attribution & Unit Economics</h2>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-dm-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              Enforced &le; $0.01 / incident ceiling
            </span>
          </div>
          <p className="text-sm text-white/50 mt-1">
            Empirical telemetry tracking of token usage, model fees, and prorated infrastructure overhead per organization.
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isBusy}
          className="flex items-center gap-2 px-3.5 py-1.5 bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.08] rounded-lg text-xs font-dm-mono text-white/80 transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isBusy ? 'animate-spin text-white/40' : ''}`} />
          Refresh Metrics
        </button>
      </div>

      {summaryError && (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-300 text-sm flex items-center gap-3">
          <ShieldAlert className="w-5 h-5 shrink-0 text-red-400" />
          <span>Failed to load tenant cost attribution metrics. Verify backend connectivity.</span>
        </div>
      )}

      {/* TOP KPI CARDS */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Card 1: Total Cost */}
        <div className="p-5 rounded-xl bg-white/[0.03] border border-white/[0.08] relative overflow-hidden flex flex-col justify-between">
          <div className="flex items-center justify-between text-white/60 mb-2">
            <span className="text-xs uppercase tracking-wider font-dm-mono">Total Monthly Cost</span>
            <DollarSign className="w-4 h-4 text-emerald-400" />
          </div>
          {summaryLoading ? (
            <Loader2 className="w-5 h-5 animate-spin text-white/30 my-2" />
          ) : (
            <div>
              <div className="text-2xl font-bold font-dm-mono text-white">
                ${summary ? summary.totalCostUsd.toFixed(4) : '0.0000'}
              </div>
              <div className="text-xs text-white/40 mt-1 font-dm-mono">
                Infra: ${summary ? summary.infraShareUsd.toFixed(2) : '15.00'} + LLM: ${summary ? summary.llmSpendUsd.toFixed(4) : '0.0000'}
              </div>
            </div>
          )}
        </div>

        {/* Card 2: Average Incident Cost */}
        <div className="p-5 rounded-xl bg-white/[0.03] border border-white/[0.08] relative overflow-hidden flex flex-col justify-between">
          <div className="flex items-center justify-between text-white/60 mb-2">
            <span className="text-xs uppercase tracking-wider font-dm-mono">Avg Cost / Incident</span>
            <Sparkles className="w-4 h-4 text-cyan-400" />
          </div>
          {summaryLoading ? (
            <Loader2 className="w-5 h-5 animate-spin text-white/30 my-2" />
          ) : (
            <div>
              <div className="text-2xl font-bold font-dm-mono text-cyan-400">
                ${summary ? summary.avgCostPerIncidentUsd.toFixed(6) : '0.000000'}
              </div>
              <div className="text-xs text-emerald-400/80 mt-1 flex items-center gap-1 font-dm-mono">
                <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                Ceiling: &le; $0.010000 (Configured)
              </div>
            </div>
          )}
        </div>

        {/* Card 3: Incidents Explained */}
        <div className="p-5 rounded-xl bg-white/[0.03] border border-white/[0.08] relative overflow-hidden flex flex-col justify-between">
          <div className="flex items-center justify-between text-white/60 mb-2">
            <span className="text-xs uppercase tracking-wider font-dm-mono">Explained Incidents</span>
            <Layers className="w-4 h-4 text-purple-400" />
          </div>
          {summaryLoading ? (
            <Loader2 className="w-5 h-5 animate-spin text-white/30 my-2" />
          ) : (
            <div>
              <div className="text-2xl font-bold font-dm-mono text-white">
                {summary ? summary.totalIncidentsExplained : 0}
              </div>
              <div className="text-xs text-white/40 mt-1 font-dm-mono">
                100% Deterministic Grounding
              </div>
            </div>
          )}
        </div>

        {/* Card 4: Storm Spend Cap */}
        <div className="p-5 rounded-xl bg-white/[0.03] border border-white/[0.08] relative overflow-hidden flex flex-col justify-between">
          <div className="flex items-center justify-between text-white/60 mb-2">
            <span className="text-xs uppercase tracking-wider font-dm-mono">Hourly Storm Cap</span>
            <Clock className="w-4 h-4 text-amber-400" />
          </div>
          {summaryLoading ? (
            <Loader2 className="w-5 h-5 animate-spin text-white/30 my-2" />
          ) : (
            <div>
              <div className="flex items-baseline justify-between">
                <span className="text-2xl font-bold font-dm-mono text-amber-400">
                  ${summary ? summary.hourlySpendUsd.toFixed(4) : '0.0000'}
                </span>
                <span className="text-xs font-dm-mono text-white/40">/ ${summary ? summary.hourlyCapUsd.toFixed(2) : '1.00'}</span>
              </div>
              <div className="w-full bg-white/[0.08] h-1.5 rounded-full mt-2 overflow-hidden">
                <div
                  className="bg-amber-400 h-full rounded-full transition-all duration-500"
                  style={{
                    width: `${Math.min(100, ((summary?.hourlySpendUsd || 0) / (summary?.hourlyCapUsd || 1)) * 100)}%`,
                  }}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* OPERATIONAL SAFETY & CALIBRATION PANELS */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Panel 1: Empirical Calibration */}
        <div className="p-5 rounded-xl bg-white/[0.02] border border-white/[0.06] flex flex-col gap-2">
          <h3 className="text-sm font-semibold text-white/90 flex items-center gap-2">
            <Layers className="w-4 h-4 text-cyan-400" />
            Infrastructure Proration & Calibration
          </h3>
          <p className="text-xs text-white/60 leading-relaxed">
            The base hosting allocation is calibrated across active tenant organizations. Step 1 establishes
            empirical tracking of container resource usage alongside LLM spend to confirm that unit economics remain
            under target margins:
          </p>
          <div className="mt-2 p-3 rounded-lg bg-black/40 border border-white/[0.06] font-dm-mono text-xs text-emerald-400">
            Tenant Share = ($15.00 Base Infra &divide; {summary?.activeOrgsCount || 1} Active Orgs) + Total Incident LLM Spend
          </div>
        </div>

        {/* Panel 2: Enforced Circuit Breakers */}
        <div className="p-5 rounded-xl bg-white/[0.02] border border-white/[0.06] flex flex-col gap-2">
          <h3 className="text-sm font-semibold text-white/90 flex items-center gap-2">
            <ShieldAlert className="w-4 h-4 text-emerald-400" />
            Two-Tier Pre-Flight & Storm Enforcement
          </h3>
          <p className="text-xs text-white/60 leading-relaxed">
            Runaway cost protection is enforced <strong>before the fact</strong>:
          </p>
          <ul className="text-xs text-white/60 space-y-1 list-disc list-inside">
            <li><strong>Pre-Flight Token Gate:</strong> Rejects calls before network dispatch if estimated cost exceeds &le; $0.01.</li>
            <li><strong>Storm Circuit Breaker:</strong> Caps hourly spend at $1.00/hr, falling back cleanly to $0 deterministic telemetry.</li>
            <li><strong>Output Validator:</strong> Blocks direct commands and soft-advisory phrasing from entering diagnoses.</li>
          </ul>
        </div>
      </div>

      {/* INCIDENT COST LEDGER TABLE */}
      <div className="flex-1 flex flex-col min-h-[250px] border border-white/[0.08] rounded-xl bg-white/[0.01] overflow-hidden">
        <div className="px-5 py-3 border-b border-white/[0.08] flex items-center justify-between bg-white/[0.02]">
          <h3 className="text-sm font-semibold text-white/90">Incident Explanation Cost Ledger</h3>
          <span className="text-xs font-dm-mono text-white/40">
            Showing {breakdown?.items?.length || 0} events
          </span>
        </div>

        <div className="flex-1 overflow-auto">
          {breakdownLoading ? (
            <div className="flex items-center justify-center h-48 text-white/40 text-sm gap-2">
              <Loader2 className="w-4 h-4 animate-spin" />
              Loading incident cost ledger...
            </div>
          ) : !breakdown?.items || breakdown.items.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-48 text-white/40 text-sm gap-2">
              <DollarSign className="w-8 h-8 text-white/20" />
              <span>No incident explanations generated yet for this organization.</span>
              <span className="text-xs text-white/30">
                Trigger on-demand diagnosis from the Anomalies tab to record cost events.
              </span>
            </div>
          ) : (
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-white/[0.06] text-white/40 font-dm-mono uppercase">
                  <th className="py-3 px-4 font-medium">Anomaly ID</th>
                  <th className="py-3 px-4 font-medium">Model</th>
                  <th className="py-3 px-4 font-medium">Prompt Tokens</th>
                  <th className="py-3 px-4 font-medium">Output Tokens</th>
                  <th className="py-3 px-4 font-medium">Estimated Cost</th>
                  <th className="py-3 px-4 font-medium">Recorded At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04] font-dm-mono">
                {breakdown.items.map((item) => (
                  <tr key={item.id} className="hover:bg-white/[0.02] transition-colors">
                    <td className="py-3 px-4 text-white/90">{item.anomalyId}</td>
                    <td className="py-3 px-4 text-cyan-400">{item.model}</td>
                    <td className="py-3 px-4 text-white/70">{item.inputTokens}</td>
                    <td className="py-3 px-4 text-white/70">{item.outputTokens}</td>
                    <td className="py-3 px-4 text-emerald-400 font-semibold">${item.estimatedCostUsd.toFixed(6)}</td>
                    <td className="py-3 px-4 text-white/40">{new Date(item.createdAt).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
};
