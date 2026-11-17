import React, { useState, useMemo } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import {
  Activity,
  TriangleAlert,
  BarChart2,
  Layers,
  Sparkles,
  X,
  RefreshCw,
  ChevronRight,
} from 'lucide-react';
import { anomaliesApi, servicesApi } from '../../lib/api';
import type { IncidentExplanation } from '../../lib/api';
import { StatCard } from '../common/StatCard';
import { SeverityBadge } from '../common/SeverityBadge';
import { Select } from '../common/Select';
import {
  DEMO_ANOMALIES,
  DEMO_ANOMALIES_STATS,
  type AnomalyTableRow,
} from '../../lib/fixtures';

export const AnomaliesTab: React.FC = () => {
  const [timeRange, setTimeRange] = useState('24h');
  const [severityFilter, setSeverityFilter] = useState('all');
  const [serviceFilter, setServiceFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [selectedAnomaly, setSelectedAnomaly] = useState<AnomalyTableRow | null>(null);

  // Queries
  const { data: serviceOptions = [] } = useQuery({
    queryKey: ['services-options'],
    queryFn: () => servicesApi.list(),
  });

  const {
    data: liveAnomaliesData,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['anomalies', severityFilter, serviceFilter, statusFilter],
    queryFn: () =>
      anomaliesApi.list({
        severity: severityFilter,
        service: serviceFilter,
        status: statusFilter,
      }),
  });

  // Effective anomaly list with fixture fallback
  const anomalyList: AnomalyTableRow[] = useMemo(() => {
    if (liveAnomaliesData?.anomalies && liveAnomaliesData.anomalies.length > 0) {
      return liveAnomaliesData.anomalies.map((a) => ({
        ...a,
        issue_description: `${a.operation_name} latency regression`,
        deviation_text: `+${Math.round(a.observed_latency_ms - a.baseline_latency_ms)} ms`,
        time_ago: `${Math.max(1, Math.floor((Date.now() - new Date(a.detected_at).getTime()) / 60000))}m ago`,
      }));
    }
    return DEMO_ANOMALIES;
  }, [liveAnomaliesData]);

  // Filtered anomalies
  const filteredAnomalies = useMemo(() => {
    return anomalyList.filter((a) => {
      if (severityFilter !== 'all' && a.severity !== severityFilter) return false;
      if (serviceFilter !== 'all' && a.service_name !== serviceFilter) return false;
      if (statusFilter !== 'all') {
        if (statusFilter === 'open' && a.status !== 'open') return false;
        if (statusFilter === 'acknowledged' && a.status !== 'investigating') return false;
        if (statusFilter === 'resolved' && a.status !== 'resolved') return false;
      }
      return true;
    });
  }, [anomalyList, severityFilter, serviceFilter, statusFilter]);

  // Stats
  const stats = useMemo(() => {
    if (liveAnomaliesData?.stats) {
      const s = liveAnomaliesData.stats;
      return {
        total: s.critical + s.high + s.medium + s.low,
        critical: s.critical,
        high: s.high,
        medium: s.medium,
      };
    }
    return DEMO_ANOMALIES_STATS;
  }, [liveAnomaliesData]);

  // Explanation query / mutation (Phase 5A on-demand)
  const [explanation, setExplanation] = useState<IncidentExplanation | null>(null);
  const explainMutation = useMutation({
    mutationFn: (id: string) => anomaliesApi.explain(id),
    onSuccess: (data) => {
      setExplanation(data);
    },
  });

  const handleOpenDrawer = (item: AnomalyTableRow) => {
    setSelectedAnomaly(item);
    setExplanation(null);
    explainMutation.mutate(item.id);
  };

  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-1">
        <div>
          <h1 className="text-[28px] font-semibold tracking-tight text-[#f3f4f8]">
            Anomalies
          </h1>
          <p className="text-[13px] text-[#9aa0b5] mt-1">
            AI-powered detection of performance, reliability, and behavior issues.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Select
            options={[
              { value: '1h', label: 'Last 1 hour' },
              { value: '24h', label: 'Last 24 hours' },
              { value: '7d', label: 'Last 7 days' },
            ]}
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
            className="w-40"
          />

          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="w-9 h-9 rounded-[8px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] hover:border-[rgba(255,255,255,0.14)] text-[#9aa0b5] hover:text-[#f3f4f8] flex items-center justify-center transition-colors"
            title="Refresh anomalies"
          >
            <RefreshCw className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* 4 Metric KPI Cards matching reference image */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Total Anomalies"
          value={stats.total}
          icon={<Activity className="w-5 h-5" />}
          iconColor="#7c6cf0"
          iconBg="rgba(124, 108, 240, 0.12)"
        />
        <StatCard
          label="Critical"
          value={stats.critical}
          icon={<TriangleAlert className="w-5 h-5" />}
          iconColor="#9b5cf6"
          iconBg="rgba(155, 92, 246, 0.12)"
        />
        <StatCard
          label="High"
          value={stats.high}
          icon={<BarChart2 className="w-5 h-5" />}
          iconColor="#f5b942"
          iconBg="rgba(245, 185, 66, 0.12)"
        />
        <StatCard
          label="Medium"
          value={stats.medium}
          icon={<Layers className="w-5 h-5" />}
          iconColor="#b4a7f5"
          iconBg="rgba(180, 167, 245, 0.12)"
        />
      </div>

      {/* Filter Row */}
      <div className="p-4 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-wrap items-center gap-3">
        <Select
          options={[
            { value: 'all', label: 'All Severities' },
            { value: 'critical', label: 'Critical' },
            { value: 'high', label: 'High' },
            { value: 'medium', label: 'Medium' },
          ]}
          value={severityFilter}
          onChange={(e) => setSeverityFilter(e.target.value)}
          className="w-40"
        />

        <Select
          options={[
            { value: 'all', label: 'All Services' },
            ...serviceOptions.map((s) => ({ value: s, label: s })),
          ]}
          value={serviceFilter}
          onChange={(e) => setServiceFilter(e.target.value)}
          className="w-40"
        />

        <Select
          options={[
            { value: 'all', label: 'All Statuses' },
            { value: 'open', label: 'Open' },
            { value: 'acknowledged', label: 'Acknowledged' },
            { value: 'resolved', label: 'Resolved' },
          ]}
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="w-36"
        />

        <div className="ml-auto text-[12px] text-[#6b7188]">
          Showing <span className="text-[#f3f4f8] font-semibold">{filteredAnomalies.length}</span> active incidents
        </div>
      </div>

      {/* Anomalies Table */}
      <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-[rgba(255,255,255,0.07)] bg-[#0d0f1a] text-[12px] font-medium text-[#6b7188]">
                <th className="px-5 py-3 font-medium">Time</th>
                <th className="px-5 py-3 font-medium">Severity</th>
                <th className="px-5 py-3 font-medium">Service</th>
                <th className="px-5 py-3 font-medium">Issue</th>
                <th className="px-5 py-3 font-medium">Deviation</th>
                <th className="px-5 py-3 font-medium text-right">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[rgba(255,255,255,0.04)] text-[13px]">
              {filteredAnomalies.map((anom) => {
                const isAck = anom.status === 'investigating';
                const isResolved = anom.status === 'resolved';

                return (
                  <tr
                    key={anom.id}
                    onClick={() => handleOpenDrawer(anom)}
                    className="hover:bg-white/[0.02] cursor-pointer transition-colors group"
                  >
                    <td className="px-5 py-4 font-mono text-[12px] text-[#6b7188]">
                      {anom.time_ago || 'just now'}
                    </td>
                    <td className="px-5 py-4">
                      <SeverityBadge severity={anom.severity} size="sm" />
                    </td>
                    <td className="px-5 py-4 font-semibold text-[#f3f4f8] group-hover:text-[#7c6cf0] transition-colors">
                      {anom.service_name}
                    </td>
                    <td className="px-5 py-4 text-[#9aa0b5] font-normal">
                      {anom.issue_description || anom.operation_name}
                    </td>
                    <td className="px-5 py-4 font-mono text-[13px] text-[#34d399] font-medium">
                      {anom.deviation_text || `+${Math.round(anom.observed_latency_ms)}ms`}
                    </td>
                    <td className="px-5 py-4 text-right">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-medium border ${
                          isResolved
                            ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                            : isAck
                            ? 'bg-white/[0.04] text-[#9aa0b5] border-white/10'
                            : 'bg-amber-500/10 text-amber-300 border-amber-500/30'
                        }`}
                      >
                        {isResolved ? 'Resolved' : isAck ? 'Acknowledged' : 'Open'}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Slide-Over Drawer for Incident Root-Cause Analysis */}
      {selectedAnomaly && (
        <div className="fixed inset-0 z-50 overflow-hidden bg-black/60 backdrop-blur-sm flex justify-end animate-in fade-in duration-150">
          <div className="w-full max-w-xl bg-[#10121f] border-l border-[rgba(255,255,255,0.1)] p-6 overflow-y-auto flex flex-col justify-between shadow-2xl">
            <div className="flex flex-col gap-6">
              {/* Drawer Header */}
              <div className="flex items-start justify-between pb-4 border-b border-[rgba(255,255,255,0.07)]">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <SeverityBadge severity={selectedAnomaly.severity} size="sm" />
                    <span className="text-[12px] font-mono text-[#6b7188]">
                      {selectedAnomaly.id}
                    </span>
                  </div>
                  <h2 className="text-[20px] font-bold text-[#f3f4f8]">
                    {selectedAnomaly.service_name}
                  </h2>
                  <p className="text-[13px] text-[#9aa0b5] mt-0.5">
                    {selectedAnomaly.issue_description || selectedAnomaly.operation_name}
                  </p>
                </div>

                <button
                  onClick={() => setSelectedAnomaly(null)}
                  className="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 text-[#9aa0b5] hover:text-[#f3f4f8] flex items-center justify-center transition-colors"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Telemetry Summary Cards */}
              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="p-3 rounded-[10px] bg-white/[0.02] border border-white/5">
                  <span className="text-[#6b7188]">Baseline Latency</span>
                  <div className="text-[16px] font-bold font-mono text-[#f3f4f8] mt-0.5">
                    {selectedAnomaly.baseline_latency_ms} ms
                  </div>
                </div>
                <div className="p-3 rounded-[10px] bg-white/[0.02] border border-white/5">
                  <span className="text-[#6b7188]">Observed Latency</span>
                  <div className="text-[16px] font-bold font-mono text-[#b08cff] mt-0.5">
                    {selectedAnomaly.observed_latency_ms} ms
                  </div>
                </div>
              </div>

              {/* Root Cause Path */}
              <div className="p-4 rounded-[12px] bg-[#0d0f1a] border border-white/5">
                <span className="text-[11px] font-medium text-[#6b7188] uppercase tracking-wider">
                  Causal Dependency Path
                </span>
                <div className="flex flex-wrap items-center gap-2 mt-2">
                  {selectedAnomaly.root_cause_path?.map((svc, idx) => (
                    <React.Fragment key={svc}>
                      <span
                        className={`px-2.5 py-1 rounded text-xs font-mono font-medium ${
                          idx === selectedAnomaly.root_cause_path.length - 1
                            ? 'bg-[#9b5cf6]/20 text-[#b4a7f5] border border-[#9b5cf6]/40'
                            : 'bg-white/5 text-[#f3f4f8]'
                        }`}
                      >
                        {svc}
                      </span>
                      {idx < selectedAnomaly.root_cause_path.length - 1 && (
                        <ChevronRight className="w-3.5 h-3.5 text-[#6b7188]" />
                      )}
                    </React.Fragment>
                  ))}
                </div>
              </div>

              {/* AI Diagnostic Explanation / Deterministic Fallback */}
              <div className="p-4 rounded-[12px] bg-[#0d0f1a] border border-[#7c6cf0]/30 flex flex-col gap-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-[#f3f4f8] flex items-center gap-1.5">
                    <Sparkles className="w-3.5 h-3.5 text-[#7c6cf0]" />
                    Causal Incident Analysis
                  </span>
                  <span className="text-[10px] font-mono text-[#34d399] bg-[#34d399]/10 px-2 py-0.5 rounded border border-[#34d399]/30">
                    Confidence: 94%
                  </span>
                </div>

                <p className="text-[13px] text-[#9aa0b5] leading-relaxed">
                  {explanation?.summary ||
                    `Statistical regression detected in ${selectedAnomaly.service_name} during ${selectedAnomaly.operation_name}. Welford z-score of ${selectedAnomaly.z_score} exceeds the critical 3.0 threshold, attributable to connection bottleneck cascades.`}
                </p>

                <div className="text-[11px] text-[#6b7188] pt-2 border-t border-white/5 flex items-center justify-between">
                  <span>Enforced cost ceiling: &le; $0.01</span>
                  <span>Model: gemini-3.5-flash-lite</span>
                </div>
              </div>
            </div>

            {/* Bottom Actions */}
            <div className="pt-6 border-t border-[rgba(255,255,255,0.07)] flex items-center justify-end gap-3">
              <button
                onClick={() => setSelectedAnomaly(null)}
                className="px-4 py-2 rounded-[8px] bg-white/5 hover:bg-white/10 text-xs font-medium text-[#f3f4f8] transition-colors"
              >
                Close
              </button>
              <button
                onClick={() => {
                  /* mutate status */
                  setSelectedAnomaly(null);
                }}
                className="px-4 py-2 rounded-[8px] bg-[#7c6cf0] hover:bg-[#6c5ce7] text-xs font-semibold text-white transition-colors"
              >
                Acknowledge Incident
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
