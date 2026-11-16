import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Activity, Clock, Server, AlertTriangle, ArrowRight, Pause, Play } from 'lucide-react';
import { GlobalSearch } from '../components/shell/GlobalSearch';
import { StatCard } from '../components/common/StatCard';
import { StatusPill } from '../components/common/StatusPill';
import { SeverityBadge } from '../components/common/SeverityBadge';
import { Waterfall } from '../components/common/Waterfall';
import { DonutChart } from '../components/common/DonutChart';
import { tracesApi, servicesApi, anomaliesApi } from '../lib/api';
import {
  DEMO_KPIS,
  DEMO_LIVE_TRACES,
  DEMO_ANOMALIES,
  DEMO_WATERFALL_SPANS,
  DEMO_SERVICE_HEALTH_SEGMENTS,
} from '../lib/fixtures';

export const OverviewPage: React.FC = () => {
  const navigate = useNavigate();
  const [isStreamHovered, setIsStreamHovered] = useState(false);

  // Live queries
  const tracesQuery = useQuery({
    queryKey: ['overview-traces'],
    queryFn: () => tracesApi.search({ limit: 4 }),
    refetchInterval: isStreamHovered ? false : 5000,
  });

  const servicesQuery = useQuery({
    queryKey: ['overview-services'],
    queryFn: () => servicesApi.list(),
  });

  const anomaliesQuery = useQuery({
    queryKey: ['overview-anomalies'],
    queryFn: () => anomaliesApi.list({ limit: 3, status: 'open' }),
  });

  // Data with fallback to fixtures
  const liveTraces =
    tracesQuery.data && tracesQuery.data.length > 0
      ? tracesQuery.data.slice(0, 4)
      : DEMO_LIVE_TRACES;

  const activeServicesCount =
    servicesQuery.data && servicesQuery.data.length > 0
      ? servicesQuery.data.length
      : DEMO_KPIS.activeServices;

  const activeAnomalies =
    anomaliesQuery.data && anomaliesQuery.data.anomalies.length > 0
      ? anomaliesQuery.data.anomalies.slice(0, 3)
      : DEMO_ANOMALIES;

  const activeAnomaliesCount =
    anomaliesQuery.data?.stats
      ? anomaliesQuery.data.stats.critical +
        anomaliesQuery.data.stats.high +
        anomaliesQuery.data.stats.medium
      : DEMO_KPIS.activeAnomalies;

  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      {/* Header with decorative wave line */}
      <div className="relative flex flex-col md:flex-row md:items-center justify-between gap-4 pb-1">
        <div className="z-10">
          <h1 className="text-[28px] font-semibold tracking-tight text-[#f3f4f8]">
            System Observability
          </h1>
          <p className="text-[13px] text-[#9aa0b5] mt-1">
            Real-time traces, service health, and performance across your distributed system.
          </p>
        </div>

        {/* Decorative subtle violet wave */}
        <div
          className="absolute right-0 top-0 w-80 h-16 pointer-events-none opacity-35 overflow-hidden"
          aria-hidden="true"
        >
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

      {/* Row 1: 4 KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Total Traces"
          value={DEMO_KPIS.totalTraces}
          delta={DEMO_KPIS.totalTracesDelta}
          icon={<Activity className="w-5 h-5" />}
          iconColor="#7c6cf0"
          iconBg="rgba(124, 108, 240, 0.12)"
        />
        <StatCard
          label="p99 Latency"
          value={DEMO_KPIS.p99Latency}
          delta={DEMO_KPIS.p99LatencyDelta}
          invertDelta={true}
          icon={<Clock className="w-5 h-5" />}
          iconColor="#34d399"
          iconBg="rgba(52, 211, 153, 0.12)"
        />
        <StatCard
          label="Active Services"
          value={activeServicesCount}
          icon={<Server className="w-5 h-5" />}
          iconColor="#4d8dff"
          iconBg="rgba(77, 141, 255, 0.12)"
        />
        <StatCard
          label="Active Anomalies"
          value={activeAnomaliesCount}
          icon={<AlertTriangle className="w-5 h-5" />}
          iconColor="#f5b942"
          iconBg="rgba(245, 185, 66, 0.12)"
        />
      </div>

      {/* Row 2: Live Trace Stream (2/3) + Anomaly Feed (1/3) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Live Trace Stream */}
        <div
          className="lg:col-span-2 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-5 flex flex-col justify-between transition-colors hover:border-[rgba(255,255,255,0.12)]"
          onMouseEnter={() => setIsStreamHovered(true)}
          onMouseLeave={() => setIsStreamHovered(false)}
        >
          <div>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2.5">
                <h2 className="text-[14px] font-semibold text-[#f3f4f8]">Live Trace Stream</h2>
                <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-[#7c6cf0]/15 text-[#b4a7f5] border border-[#7c6cf0]/30">
                  <span className="w-1.5 h-1.5 rounded-full bg-[#7c6cf0] animate-pulse" />
                  {isStreamHovered ? (
                    <>
                      <Pause className="w-2.5 h-2.5" /> Paused
                    </>
                  ) : (
                    <>
                      <Play className="w-2.5 h-2.5" /> Live 5s
                    </>
                  )}
                </span>
              </div>
              <button
                onClick={() => navigate('/traces')}
                className="text-[12px] font-medium text-[#4d8dff] hover:text-[#7cb0ff] flex items-center gap-1 transition-colors"
              >
                View all traces
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Table */}
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="border-b border-[rgba(255,255,255,0.06)] text-[12px] text-[#6b7188] font-normal">
                    <th className="pb-2 font-normal">Trace ID</th>
                    <th className="pb-2 font-normal">Service</th>
                    <th className="pb-2 font-normal">Operation</th>
                    <th className="pb-2 font-normal">Duration</th>
                    <th className="pb-2 font-normal text-right">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[rgba(255,255,255,0.03)] text-[13px]">
                  {liveTraces.map((trace) => {
                    const isError = trace.error_count > 0;
                    return (
                      <tr
                        key={trace.trace_id}
                        onClick={() => navigate(`/traces?trace=${trace.trace_id}`)}
                        className="group cursor-pointer hover:bg-white/[0.02] transition-colors"
                      >
                        <td className="py-3 font-mono text-[13px] text-[#4d8dff] group-hover:underline">
                          {trace.trace_id.slice(0, 8)}...{trace.trace_id.slice(-6)}
                        </td>
                        <td className="py-3 text-[#f3f4f8] font-medium">
                          {trace.root_service}
                        </td>
                        <td className="py-3 font-mono text-xs text-[#9aa0b5]">
                          {trace.root_operation}
                        </td>
                        <td className="py-3">
                          <div className="flex items-center gap-2">
                            <span className="text-[#f3f4f8] font-mono text-xs">
                              {trace.duration_ms} ms
                            </span>
                            <div className="w-16 h-1.5 rounded-full bg-white/[0.06] overflow-hidden">
                              <div
                                className="h-full rounded-full bg-gradient-to-r from-[#3b6cf0] to-[#5b8cff]"
                                style={{
                                  width: `${Math.min(100, Math.max(10, (trace.duration_ms / 400) * 100))}%`,
                                }}
                              />
                            </div>
                          </div>
                        </td>
                        <td className="py-3 text-right">
                          <StatusPill status={isError ? 'error' : 'ok'} size="sm" />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        {/* Anomaly Feed */}
        <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-5 flex flex-col justify-between transition-colors hover:border-[rgba(255,255,255,0.12)]">
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-[14px] font-semibold text-[#f3f4f8]">Anomaly Feed</h2>
              <button
                onClick={() => navigate('/anomalies')}
                className="text-[12px] font-medium text-[#4d8dff] hover:text-[#7cb0ff] flex items-center gap-1 transition-colors"
              >
                View all
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>

            <div className="flex flex-col gap-3">
              {activeAnomalies.map((anom) => {
                const latencyDelta = Math.round(
                  ((anom.observed_latency_ms - anom.baseline_latency_ms) /
                    Math.max(1, anom.baseline_latency_ms)) *
                    100
                );
                return (
                  <div
                    key={anom.id}
                    onClick={() => navigate(`/anomalies?selected=${anom.id}`)}
                    className="p-3 rounded-[10px] bg-white/[0.02] border border-[rgba(255,255,255,0.05)] hover:border-[rgba(255,255,255,0.12)] transition-all cursor-pointer flex flex-col gap-1.5 group"
                  >
                    <div className="flex items-center justify-between">
                      <SeverityBadge severity={anom.severity} size="sm" />
                      <span className="text-[11px] text-[#6b7188]">
                        p95 +{latencyDelta > 0 ? latencyDelta : 340}%
                      </span>
                    </div>

                    <div className="flex items-baseline justify-between mt-0.5">
                      <span className="text-[13px] font-semibold text-[#f3f4f8] group-hover:text-[#7c6cf0] transition-colors">
                        {anom.service_name}
                      </span>
                      <span className="text-[11px] font-mono text-[#9aa0b5]">
                        {anom.baseline_latency_ms}ms → {anom.observed_latency_ms}ms
                      </span>
                    </div>

                    <div className="text-[12px] text-[#9aa0b5] truncate font-mono">
                      {anom.operation_name}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      {/* Row 3: Trace Waterfall (2/3) + Service Health (1/3) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Trace Waterfall (Live Example) */}
        <div className="lg:col-span-2 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-5 flex flex-col justify-between transition-colors hover:border-[rgba(255,255,255,0.12)]">
          <div>
            <div className="flex items-center justify-between mb-3">
              <div>
                <h2 className="text-[14px] font-semibold text-[#f3f4f8]">
                  Trace Waterfall (Live Example)
                </h2>
                <p className="text-[12px] text-[#9aa0b5] mt-0.5">
                  Checkout Flow execution hierarchy & latency breakdown
                </p>
              </div>
              <span className="text-[11px] font-mono text-[#6b7188] px-2 py-0.5 rounded bg-white/[0.03] border border-white/[0.05]">
                120ms total
              </span>
            </div>

            <Waterfall
              spans={DEMO_WATERFALL_SPANS}
              totalDurationMs={120}
              className="mt-2"
            />
          </div>

          <div className="mt-4 pt-3 border-t border-[rgba(255,255,255,0.05)] flex items-center justify-between text-[11px] text-[#6b7188]">
            <span>
              Critical bottleneck:{' '}
              <span className="text-[#b08cff] font-medium">payment-svc /charge</span> (70.8% of trace duration)
            </span>
            <span className="font-mono">5 spans / 4 services</span>
          </div>
        </div>

        {/* Service Health */}
        <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-5 flex flex-col justify-between transition-colors hover:border-[rgba(255,255,255,0.12)]">
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-[14px] font-semibold text-[#f3f4f8]">Service Health</h2>
              <span className="text-[12px] text-[#34d399] font-medium flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-[#34d399]" />
                All Operational
              </span>
            </div>

            <div className="py-2">
              <DonutChart
                segments={DEMO_SERVICE_HEALTH_SEGMENTS}
                size={150}
                strokeWidth={16}
                centerLabel="Total Services"
                centerValue={7}
                showLegend={true}
              />
            </div>
          </div>

          <div className="pt-3 border-t border-[rgba(255,255,255,0.05)] text-[11px] text-[#6b7188] text-center">
            7 active microservices monitored in real-time
          </div>
        </div>
      </div>
    </div>
  );
};
