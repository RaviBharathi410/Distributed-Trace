import React, { useState, useMemo } from 'react';
import { AlertCircle, AlertTriangle, Info, CheckCircle2, Search, Filter, RefreshCw, Loader2, Sparkles, ShieldAlert } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceArea } from 'recharts';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { anomaliesApi } from '../../lib/api';
import type { Anomaly, IncidentExplanation } from '../../lib/api';

const generateChartData = (baseline: number, observed: number) => {
  return Array.from({ length: 60 }).map((_, i) => {
    const isAnomalyWindow = i > 40 && i < 52;
    const val = isAnomalyWindow ? observed : baseline + (Math.random() * baseline * 0.15);
    return {
      time: `-${60 - i}m`,
      value: Math.round(val),
    };
  });
};

export const AnomaliesTab: React.FC = () => {
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState('');
  const [severityFilter, setSeverityFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [explanations, setExplanations] = useState<Record<string, IncidentExplanation>>({});

  // Real React Query: Fetch anomalies list and stats from ClickHouse backend
  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['anomalies', severityFilter, statusFilter],
    queryFn: () => anomaliesApi.list({ severity: severityFilter, status: statusFilter }),
  });

  const anomalies: Anomaly[] = useMemo(() => data?.anomalies || [], [data]);
  const stats = useMemo(() => data?.stats || { critical: 0, high: 0, medium: 0, low: 0 }, [data]);

  // Mutation for updating status
  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'open' | 'investigating' | 'resolved' }) =>
      anomaliesApi.updateStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
    },
  });

  // Mutation for on-demand root cause explanation (Phase 5A)
  const explainMutation = useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) => anomaliesApi.explain(id, force),
    onSuccess: (result, variables) => {
      setExplanations((prev) => ({ ...prev, [variables.id]: result }));
    },
  });

  const formatCost = (val: number) => `$${val.toFixed(6)}`;

  const filtered = useMemo(() => {
    let res = [...anomalies];
    if (search) {
      res = res.filter(
        (a) =>
          a.id.toLowerCase().includes(search.toLowerCase()) ||
          a.service_name.toLowerCase().includes(search.toLowerCase()) ||
          a.operation_name.toLowerCase().includes(search.toLowerCase()),
      );
    }
    return res;
  }, [anomalies, search]);

  const selectedAnomaly = useMemo(() => filtered.find((a) => a.id === selectedId), [selectedId, filtered]);

  const chartData = useMemo(() => {
    if (!selectedAnomaly) return [];
    return generateChartData(selectedAnomaly.baseline_latency_ms || 50, selectedAnomaly.observed_latency_ms || 250);
  }, [selectedAnomaly]);

  const toggleRow = (id: string, checked: boolean) => {
    const next = new Set(selectedRows);
    if (checked) next.add(id);
    else next.delete(id);
    setSelectedRows(next);
  };

  const toggleAll = (checked: boolean) => {
    if (checked) setSelectedRows(new Set(filtered.map((a) => a.id)));
    else setSelectedRows(new Set());
  };

  const getSeverityColor = (sev: string) => {
    if (sev === 'critical') return 'text-red-500 bg-red-500/10 border-red-500/20';
    if (sev === 'high') return 'text-orange-500 bg-orange-500/10 border-orange-500/20';
    if (sev === 'medium') return 'text-amber-500 bg-amber-500/10 border-amber-500/20';
    return 'text-blue-500 bg-blue-500/10 border-blue-500/20';
  };

  const getSeverityIcon = (sev: string) => {
    if (sev === 'critical') return <AlertCircle className="w-3 h-3" />;
    if (sev === 'high' || sev === 'medium') return <AlertTriangle className="w-3 h-3" />;
    return <Info className="w-3 h-3" />;
  };

  const getStatusColor = (status: string) => {
    if (status === 'resolved') return 'text-green-400';
    if (status === 'investigating') return 'text-amber-400';
    return 'text-red-400';
  };

  const activeCount = stats.critical + stats.high + stats.medium + stats.low;

  return (
    <div className="flex flex-col h-full bg-black font-dm-sans text-white">
      {/* STATS STRIP */}
      <div className="grid grid-cols-3 gap-6 mb-6 shrink-0">
        <div className="bg-white/[0.02] border border-white/[0.08] rounded-lg p-5 flex flex-col gap-1">
          <span className="text-sm text-white/50 uppercase tracking-wider">Total Active Anomalies</span>
          <span className="text-3xl font-bold font-dm-mono">{activeCount}</span>
        </div>
        <div className="bg-red-500/5 border border-red-500/20 rounded-lg p-5 flex flex-col gap-1 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-16 h-16 bg-red-500/10 rounded-bl-full" />
          <span className="text-sm text-red-400/80 uppercase tracking-wider">Critical Severity</span>
          <span className="text-3xl font-bold text-red-400 font-dm-mono">{stats.critical}</span>
        </div>
        <div className="bg-white/[0.02] border border-white/[0.08] rounded-lg p-5 flex flex-col gap-1">
          <span className="text-sm text-white/50 uppercase tracking-wider">High & Medium Severity</span>
          <span className="text-3xl font-bold font-dm-mono">{stats.high + stats.medium}</span>
        </div>
      </div>

      {/* ERROR BANNER */}
      {isError && (
        <div className="mb-4 p-4 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-between text-xs">
          <div className="flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-amber-400 shrink-0" />
            <span>
              Backend anomaly service unreachable:{' '}
              <span className="text-amber-300">{(error as Error)?.message || 'http://localhost:8080'}</span>
            </span>
          </div>
          <button
            onClick={() => refetch()}
            className="text-xs bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 px-3 py-1.5 rounded border border-amber-500/40 transition-colors"
          >
            Retry Connection
          </button>
        </div>
      )}

      <div className="flex flex-1 min-h-0 gap-6">
        {/* LIST AREA */}
        <div
          className={`flex flex-col bg-white/[0.02] border border-white/[0.08] rounded-lg overflow-hidden transition-all duration-300 ${
            selectedId ? 'w-[50%]' : 'w-full'
          }`}
        >
          <div className="p-4 border-b border-white/[0.08] flex items-center justify-between bg-black/50">
            <div className="flex gap-3 items-center">
              <div className="flex items-center gap-2">
                <Filter className="w-4 h-4 text-white/40" />
                <select
                  value={severityFilter}
                  onChange={(e) => setSeverityFilter(e.target.value)}
                  className="bg-white/[0.04] border border-white/10 rounded px-2.5 py-1 text-xs text-white"
                >
                  <option value="all">All Severities</option>
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="bg-white/[0.04] border border-white/10 rounded px-2.5 py-1 text-xs text-white"
                >
                  <option value="all">All Statuses</option>
                  <option value="open">Open</option>
                  <option value="investigating">Investigating</option>
                  <option value="resolved">Resolved</option>
                </select>
              </div>

              <button
                onClick={() => refetch()}
                disabled={isFetching}
                className="p-1.5 hover:bg-white/10 rounded text-white/50 hover:text-white"
                title="Refresh"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
              </button>
            </div>
            <div className="relative w-64">
              <Search className="absolute left-3 top-2 w-4 h-4 text-white/40" />
              <input
                type="text"
                placeholder="Search anomalies..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full bg-white/[0.05] border border-white/10 rounded-md pl-9 pr-3 py-1.5 text-sm text-white placeholder:text-white/30 focus:outline-none focus:border-white/30"
              />
            </div>
          </div>
          <div className="flex-1 overflow-auto">
            {isLoading ? (
              <div className="flex flex-col items-center justify-center h-48 text-white/40 gap-2">
                <Loader2 className="w-6 h-6 animate-spin text-blue-400" />
                <span className="text-sm">Querying anomalies from ClickHouse...</span>
              </div>
            ) : filtered.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-48 text-white/40 gap-2">
                <CheckCircle2 className="w-8 h-8 opacity-40 text-emerald-400" />
                <span className="text-sm">No anomalies found matching current filters.</span>
              </div>
            ) : (
              <table className="w-full text-left">
                <thead className="sticky top-0 bg-[#0a0a0a] border-b border-white/[0.08] z-10">
                  <tr>
                    <th className="px-4 py-3 w-10">
                      <input
                        type="checkbox"
                        checked={selectedRows.size > 0 && selectedRows.size === filtered.length}
                        onChange={(e) => toggleAll(e.target.checked)}
                        className="accent-blue-500 cursor-pointer"
                      />
                    </th>
                    {['Severity', 'Service / Operation', 'Observed vs Base', 'Status'].map((col) => (
                      <th key={col} className="px-4 py-3 text-[11px] uppercase tracking-wider text-white/40 font-medium">
                        {col}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/[0.04]">
                  {filtered.map((anomaly) => (
                    <tr
                      key={anomaly.id}
                      onClick={() => setSelectedId(selectedId === anomaly.id ? null : anomaly.id)}
                      className={`hover:bg-white/[0.04] cursor-pointer transition-colors ${
                        selectedId === anomaly.id ? 'bg-white/[0.06]' : ''
                      }`}
                    >
                      <td className="px-4 py-4" onClick={(e) => e.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={selectedRows.has(anomaly.id)}
                          onChange={(e) => toggleRow(anomaly.id, e.target.checked)}
                          className="accent-blue-500 cursor-pointer"
                        />
                      </td>
                      <td className="px-4 py-4">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs uppercase tracking-wider border font-bold ${getSeverityColor(
                            anomaly.severity,
                          )}`}
                        >
                          {getSeverityIcon(anomaly.severity)}
                          {anomaly.severity}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex flex-col gap-0.5">
                          <span className="font-medium text-white/90">{anomaly.service_name}</span>
                          <span className="text-xs text-white/50">{anomaly.operation_name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-4 font-dm-mono text-xs text-white/70">
                        {Math.round(anomaly.observed_latency_ms)}ms / {Math.round(anomaly.baseline_latency_ms)}ms
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex flex-col gap-1">
                          <span
                            className={`text-sm capitalize font-medium flex items-center gap-1.5 ${getStatusColor(
                              anomaly.status,
                            )}`}
                          >
                            {anomaly.status === 'resolved' && <CheckCircle2 className="w-4 h-4" />}
                            {anomaly.status === 'investigating' && <Search className="w-4 h-4" />}
                            {anomaly.status === 'open' && <AlertCircle className="w-4 h-4" />}
                            {anomaly.status}
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        {/* DETAILS PANEL */}
        {selectedAnomaly && (
          <div className="w-[50%] bg-white/[0.02] border border-white/[0.08] rounded-lg overflow-y-auto flex flex-col animate-in slide-in-from-right-8 duration-300">
            <div className="p-6 border-b border-white/[0.08] flex justify-between items-start sticky top-0 bg-[#0c0c0c] z-10">
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-3">
                  <span
                    className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs uppercase tracking-wider border font-bold ${getSeverityColor(
                      selectedAnomaly.severity,
                    )}`}
                  >
                    {getSeverityIcon(selectedAnomaly.severity)} {selectedAnomaly.severity}
                  </span>
                  <span className="font-dm-mono text-white/50 text-sm">{selectedAnomaly.id}</span>
                </div>
                <h2 className="text-2xl font-bold mt-1">{selectedAnomaly.service_name}</h2>
                <span className="text-white/60">{selectedAnomaly.operation_name}</span>
              </div>
              <div className="flex gap-2">
                {selectedAnomaly.status !== 'investigating' && (
                  <button
                    onClick={() => updateStatusMutation.mutate({ id: selectedAnomaly.id, status: 'investigating' })}
                    disabled={updateStatusMutation.isPending}
                    className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-sm rounded transition-colors"
                  >
                    Investigate
                  </button>
                )}
                {selectedAnomaly.status !== 'resolved' && (
                  <button
                    onClick={() => updateStatusMutation.mutate({ id: selectedAnomaly.id, status: 'resolved' })}
                    disabled={updateStatusMutation.isPending}
                    className="px-4 py-2 bg-green-500/20 hover:bg-green-500/30 text-green-400 border border-green-500/30 text-sm rounded transition-colors"
                  >
                    Resolve
                  </button>
                )}
              </div>
            </div>

            <div className="p-6 flex flex-col gap-8">
              {/* CHART */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm font-semibold uppercase tracking-wider text-white/50">Telemetry Context (Latency Profile)</h3>
                <div className="h-48 w-full bg-black/50 border border-white/[0.05] rounded p-2">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={chartData}>
                      <XAxis dataKey="time" stroke="rgba(255,255,255,0.2)" fontSize={10} />
                      <YAxis stroke="rgba(255,255,255,0.2)" fontSize={10} width={35} />
                      <Tooltip contentStyle={{ backgroundColor: '#111', border: '1px solid rgba(255,255,255,0.1)' }} />
                      <ReferenceArea x1="-20m" x2="-8m" fill="rgba(239, 68, 68, 0.15)" strokeOpacity={0.5} />
                      <Line type="monotone" dataKey="value" stroke="#3b82f6" strokeWidth={2} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* STATISTICAL DEVIATION */}
              <div className="grid grid-cols-3 gap-3">
                <div className="p-3 bg-white/[0.03] border border-white/[0.05] rounded">
                  <span className="text-[10px] text-white/50 uppercase block">Observed Latency</span>
                  <span className="text-base font-bold font-dm-mono text-red-400">{Math.round(selectedAnomaly.observed_latency_ms)}ms</span>
                </div>
                <div className="p-3 bg-white/[0.03] border border-white/[0.05] rounded">
                  <span className="text-[10px] text-white/50 uppercase block">Baseline Latency</span>
                  <span className="text-base font-bold font-dm-mono text-white/70">{Math.round(selectedAnomaly.baseline_latency_ms)}ms</span>
                </div>
                <div className="p-3 bg-white/[0.03] border border-white/[0.05] rounded">
                  <span className="text-[10px] text-white/50 uppercase block">Z-Score Deviation</span>
                  <span className="text-base font-bold font-dm-mono text-amber-400">{selectedAnomaly.z_score.toFixed(2)}σ</span>
                </div>
              </div>

              {/* ROOT CAUSE PATH */}
              {selectedAnomaly.root_cause_path && selectedAnomaly.root_cause_path.length > 0 && (
                <div className="flex flex-col gap-3">
                  <h3 className="text-sm font-semibold uppercase tracking-wider text-white/50">Root Cause Propagation Path</h3>
                  <div className="flex items-center gap-2 p-3 bg-white/[0.03] border border-white/[0.05] rounded font-dm-mono text-xs">
                    {selectedAnomaly.root_cause_path.map((step, idx) => (
                      <React.Fragment key={idx}>
                        <span className={idx === selectedAnomaly.root_cause_path.length - 1 ? 'text-red-400 font-bold' : 'text-white/70'}>
                          {step}
                        </span>
                        {idx < selectedAnomaly.root_cause_path.length - 1 && <span className="text-white/30">→</span>}
                      </React.Fragment>
                    ))}
                  </div>
                </div>
              )}

              {/* ROOT CAUSE EXPLANATION (Phase 5A) */}
              {(() => {
                const currentExp = selectedAnomaly ? explanations[selectedAnomaly.id] : undefined;
                const isExplaining = explainMutation.isPending && explainMutation.variables?.id === selectedAnomaly.id;

                return (
                  <div className="flex flex-col gap-3 pt-3 border-t border-white/[0.08]">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Sparkles className="w-4 h-4 text-purple-400" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-white/80">
                          Root Cause Explanation & Diagnostics
                        </h3>
                      </div>
                      {currentExp && (
                        <button
                          onClick={() => explainMutation.mutate({ id: selectedAnomaly.id, force: true })}
                          disabled={explainMutation.isPending}
                          className="text-xs text-white/50 hover:text-white flex items-center gap-1.5 transition-colors disabled:opacity-50"
                          title="Re-run explanation (force cache refresh)"
                        >
                          <RefreshCw className={`w-3 h-3 ${isExplaining ? 'animate-spin' : ''}`} />
                          <span>Re-analyze</span>
                        </button>
                      )}
                    </div>

                    {/* STATE 1: UNANALYZED */}
                    {!currentExp && !isExplaining && (
                      <div className="p-4 bg-purple-950/10 border border-purple-500/20 rounded-lg flex flex-col gap-3">
                        <div className="flex flex-col gap-1">
                          <span className="text-sm font-medium text-purple-200">On-Demand Root Cause Analysis</span>
                          <p className="text-xs text-white/60">
                            Synthesizes Phase 3 deterministic telemetry and critical-path traversal into a grounded diagnostic narrative.
                            Confined to descriptive telemetry observations with zero remediation directives.
                          </p>
                        </div>

                        <div className="flex items-center justify-between pt-2 border-t border-purple-500/10">
                          <button
                            onClick={() => explainMutation.mutate({ id: selectedAnomaly.id, force: false })}
                            className="px-3.5 py-1.5 bg-purple-600 hover:bg-purple-500 text-white text-xs font-semibold rounded flex items-center gap-2 transition-all shadow-sm shadow-purple-900/50"
                          >
                            <Sparkles className="w-3.5 h-3.5" />
                            <span>Explain Root Cause with AI (≤ $0.01)</span>
                          </button>
                          <span className="text-[11px] text-white/40 font-dm-mono">
                            Ceiling: ≤ $0.010000 · Storm Cap: $1.00/hr
                          </span>
                        </div>

                        {explainMutation.isError && (
                          <div className="p-2.5 bg-red-500/10 border border-red-500/20 rounded text-xs text-red-400 flex items-center gap-2">
                            <AlertCircle className="w-3.5 h-3.5 shrink-0" />
                            <span>{explainMutation.error?.message || 'Failed to explain incident'}</span>
                          </div>
                        )}
                      </div>
                    )}

                    {/* LOADING STATE */}
                    {isExplaining && (
                      <div className="p-6 bg-purple-950/10 border border-purple-500/20 rounded-lg flex flex-col items-center justify-center gap-3">
                        <Loader2 className="w-6 h-6 text-purple-400 animate-spin" />
                        <div className="flex flex-col items-center gap-0.5 text-center">
                          <span className="text-xs font-medium text-purple-200">Grounding Telemetry & Evaluating Safety Rules...</span>
                          <span className="text-[11px] text-white/50">Running pre-flight ceiling checks and two-tier output validation</span>
                        </div>
                      </div>
                    )}

                    {/* STATE 2: AI PROSE DIAGNOSIS (COMPLIANT) */}
                    {currentExp && !currentExp.degraded_to_deterministic && (
                      <div className="p-4 bg-purple-950/15 border border-purple-500/30 rounded-lg flex flex-col gap-4">
                        <div className="flex items-center justify-between pb-3 border-b border-purple-500/20">
                          <div className="flex items-center gap-2">
                            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[11px] font-semibold bg-purple-500/20 text-purple-300 border border-purple-500/30">
                              <Sparkles className="w-3 h-3 text-purple-400" />
                              AI Root Cause Diagnosis ({currentExp.cost_attribution.model})
                            </span>
                            <span className="text-xs font-dm-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
                              {(currentExp.confidence_score * 100).toFixed(0)}% Confidence
                            </span>
                          </div>

                          <div className="flex items-center gap-2">
                            {currentExp.cost_attribution.cached ? (
                              <span className="text-[11px] font-dm-mono text-cyan-400 bg-cyan-500/10 px-2 py-0.5 rounded border border-cyan-500/20">
                                ⚡ Cached ($0.000000)
                              </span>
                            ) : (
                              <span className="text-[11px] font-dm-mono text-purple-300 bg-purple-500/10 px-2 py-0.5 rounded border border-purple-500/20">
                                {formatCost(currentExp.cost_attribution.estimated_cost_usd)} · {currentExp.cost_attribution.input_tokens + currentExp.cost_attribution.output_tokens} tokens
                              </span>
                            )}
                          </div>
                        </div>

                        <div className="flex flex-col gap-2">
                          <span className="text-xs font-semibold text-white/50 uppercase tracking-wider">Diagnostic Summary</span>
                          <p className="text-sm text-white/90 leading-relaxed font-sans bg-black/30 p-3 rounded border border-white/[0.05]">
                            {currentExp.summary}
                          </p>
                        </div>

                        {currentExp.contributing_factors && currentExp.contributing_factors.length > 0 && (
                          <div className="flex flex-col gap-2">
                            <span className="text-xs font-semibold text-white/50 uppercase tracking-wider">Contributing Telemetry Factors</span>
                            <div className="flex flex-col gap-1.5">
                              {currentExp.contributing_factors.map((factor, idx) => (
                                <div key={idx} className="flex items-start gap-2 p-2 bg-white/[0.02] border border-white/[0.05] rounded text-xs text-white/80">
                                  <span className="text-purple-400 font-bold shrink-0">•</span>
                                  <span>{factor}</span>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        <div className="pt-2 border-t border-purple-500/20 flex items-center justify-between text-[11px] text-white/40 font-dm-mono">
                          <span>Ceiling: ≤ $0.010000 (Enforced)</span>
                          <span>Hourly Spend: ${currentExp.cost_attribution.hourly_spend_usd?.toFixed(4) || '0.0000'} / ${currentExp.cost_attribution.hourly_cap_usd?.toFixed(2) || '1.00'}</span>
                        </div>
                      </div>
                    )}

                    {/* STATE 3: DEGRADED FALLBACK (HONEST TRANSPARENCY) */}
                    {currentExp && currentExp.degraded_to_deterministic && (
                      <div className="p-4 bg-amber-950/15 border border-amber-500/30 rounded-lg flex flex-col gap-4">
                        <div className="flex items-center justify-between pb-3 border-b border-amber-500/20">
                          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30">
                            <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
                            Fallback: Deterministic Telemetry Diagnosis
                          </span>

                          {currentExp.cost_attribution.input_tokens > 0 ? (
                            <span className="text-[11px] font-dm-mono text-amber-300 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20" title="Token charge logged to tenant ledger before validator rejected output">
                              Rejected by Validator · {formatCost(currentExp.cost_attribution.estimated_cost_usd)} ({currentExp.cost_attribution.input_tokens + currentExp.cost_attribution.output_tokens} tokens)
                            </span>
                          ) : (
                            <span className="text-[11px] font-dm-mono text-green-400 bg-green-500/10 px-2 py-0.5 rounded border border-green-500/20">
                              $0.000000 LLM Incurred
                            </span>
                          )}
                        </div>

                        {/* Prominent Transparency Banner */}
                        <div className="p-3 bg-amber-500/10 border border-amber-500/25 rounded flex items-start gap-2.5">
                          <ShieldAlert className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                          <div className="flex flex-col gap-0.5">
                            <span className="text-xs font-semibold text-amber-300">
                              AI Explanation Bypassed — Serving Verified Mathematical Telemetry
                            </span>
                            <p className="text-[11px] text-amber-200/80 leading-normal">
                              {currentExp.fallback_reason || 'Safety validator or pre-flight cost limit triggered.'}
                            </p>
                          </div>
                        </div>

                        {/* Mathematical Telemetry Grid */}
                        <div className="grid grid-cols-2 gap-2">
                          <div className="p-2.5 bg-black/40 border border-white/[0.05] rounded flex flex-col gap-0.5">
                            <span className="text-[10px] text-white/50 uppercase font-semibold">Isolated Bottleneck Service</span>
                            <span className="text-xs font-bold font-dm-mono text-red-400">{currentExp.root_cause_service}</span>
                            <span className="text-[10px] text-white/40">{currentExp.operation_name}</span>
                          </div>
                          <div className="p-2.5 bg-black/40 border border-white/[0.05] rounded flex flex-col gap-0.5">
                            <span className="text-[10px] text-white/50 uppercase font-semibold">Latency vs Welford Baseline</span>
                            <span className="text-xs font-bold font-dm-mono text-amber-400">
                              {Math.round(selectedAnomaly.observed_latency_ms)}ms (vs {Math.round(selectedAnomaly.baseline_latency_ms)}ms)
                            </span>
                            <span className="text-[10px] text-white/40">Z-Score: {selectedAnomaly.z_score.toFixed(2)}σ above normal</span>
                          </div>
                        </div>

                        {/* Descriptive Telemetry Observation */}
                        <div className="flex flex-col gap-2">
                          <span className="text-xs font-semibold text-white/50 uppercase tracking-wider">Deterministic Findings</span>
                          <p className="text-xs text-white/90 leading-relaxed font-sans bg-black/30 p-3 rounded border border-white/[0.05]">
                            {currentExp.summary}
                          </p>
                        </div>

                        {currentExp.contributing_factors && currentExp.contributing_factors.length > 0 && (
                          <div className="flex flex-col gap-2">
                            <span className="text-xs font-semibold text-white/50 uppercase tracking-wider">Verified Telemetry Facts</span>
                            <div className="flex flex-col gap-1.5">
                              {currentExp.contributing_factors.map((factor, idx) => (
                                <div key={idx} className="flex items-start gap-2 p-2 bg-white/[0.02] border border-white/[0.05] rounded text-xs text-white/80">
                                  <span className="text-amber-400 font-bold shrink-0">•</span>
                                  <span>{factor}</span>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        <div className="pt-2 border-t border-amber-500/20 flex items-center justify-between text-[11px] text-white/40 font-dm-mono">
                          <span>Deterministic Fallback Mode ($0 Incremental Fee)</span>
                          <span>Hourly Spend: ${currentExp.cost_attribution.hourly_spend_usd?.toFixed(4) || '0.0000'}</span>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
