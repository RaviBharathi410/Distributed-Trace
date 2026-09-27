import React, { useState, useMemo } from 'react';
import { AlertCircle, AlertTriangle, Info, CheckCircle2, Search, Filter, RefreshCw, Loader2 } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceArea } from 'recharts';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { anomaliesApi } from '../../lib/api';
import type { Anomaly } from '../../lib/api';

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
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
