import React, { useState, useMemo } from 'react';
import { AlertCircle, AlertTriangle, Info, Clock, CheckCircle2, Search, Filter } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceArea } from 'recharts';

interface Anomaly {
  id: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  service: string;
  operation: string;
  type: 'latency_spike' | 'error_rate' | 'throughput_drop';
  detectedTime: number;
  duration: string;
  delta: string;
  status: 'open' | 'investigating' | 'resolved';
  hypothesis: string;
}

const mockAnomalies: Anomaly[] = [
  {
    id: 'ANM-9021', severity: 'critical', service: 'payment-svc', operation: 'charge_card',
    type: 'error_rate', detectedTime: Date.now() - 1000 * 60 * 5, duration: 'Ongoing',
    delta: '+15.2% vs baseline', status: 'open',
    hypothesis: 'Database connection pool exhausted due to sudden spike in concurrent requests from checkout-api. Upstream timeouts observed.'
  },
  {
    id: 'ANM-9020', severity: 'high', service: 'inventory-svc', operation: 'reserve_items',
    type: 'latency_spike', detectedTime: Date.now() - 1000 * 60 * 25, duration: '12m',
    delta: '+850ms vs baseline', status: 'investigating',
    hypothesis: 'Redis cache eviction rate spiked, leading to database fallback and increased latency on stock checks.'
  },
  {
    id: 'ANM-9019', severity: 'medium', service: 'checkout-api', operation: 'process_checkout',
    type: 'throughput_drop', detectedTime: Date.now() - 1000 * 60 * 60 * 2, duration: '5m',
    delta: '-40% vs baseline', status: 'resolved',
    hypothesis: 'Brief network partition in us-east-1 availability zone causing dropped packets between API gateway and checkout-api.'
  }
];

const generateChartData = (type: string) => {
  return Array.from({ length: 60 }).map((_, i) => {
    const isAnomalyWindow = i > 40 && i < 50;
    const baseVal = type === 'latency_spike' ? 50 : 2;
    const spike = type === 'throughput_drop' ? -40 : type === 'latency_spike' ? 800 : 15;
    
    return {
      time: `-${60 - i}m`,
      value: baseVal + (isAnomalyWindow ? spike : (Math.random() * baseVal * 0.2)),
    };
  });
};

export const AnomaliesTab: React.FC = () => {
  const [anomalies, setAnomalies] = useState(mockAnomalies);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());

  const selectedAnomaly = useMemo(() => anomalies.find(a => a.id === selectedId), [selectedId, anomalies]);
  const chartData = useMemo(() => selectedAnomaly ? generateChartData(selectedAnomaly.type) : [], [selectedAnomaly]);

  const toggleRow = (id: string, checked: boolean) => {
    const next = new Set(selectedRows);
    if (checked) next.add(id);
    else next.delete(id);
    setSelectedRows(next);
  };

  const toggleAll = (checked: boolean) => {
    if (checked) setSelectedRows(new Set(anomalies.map(a => a.id)));
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

  const updateStatus = (status: 'open' | 'investigating' | 'resolved') => {
    if (selectedId) {
      setAnomalies(anomalies.map(a => a.id === selectedId ? { ...a, status } : a));
    }
  };

  return (
    <div className="flex flex-col h-full bg-black font-dm-sans">
      {/* STATS STRIP */}
      <div className="grid grid-cols-3 gap-6 mb-6 shrink-0">
        <div className="bg-white/[0.02] border border-white/[0.08] rounded-lg p-5 flex flex-col gap-1">
          <span className="text-sm text-white/50 uppercase tracking-wider">Total (Last 24h)</span>
          <span className="text-3xl font-bold">124</span>
        </div>
        <div className="bg-red-500/5 border border-red-500/20 rounded-lg p-5 flex flex-col gap-1 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-16 h-16 bg-red-500/10 rounded-bl-full" />
          <span className="text-sm text-red-400/80 uppercase tracking-wider">Active Anomalies</span>
          <span className="text-3xl font-bold text-red-400">3</span>
        </div>
        <div className="bg-white/[0.02] border border-white/[0.08] rounded-lg p-5 flex flex-col gap-1">
          <span className="text-sm text-white/50 uppercase tracking-wider">Mean Time To Detect</span>
          <span className="text-3xl font-bold font-dm-mono">42s</span>
        </div>
      </div>

      <div className="flex flex-1 min-h-0 gap-6">
        {/* LIST AREA */}
        <div className={`flex flex-col bg-white/[0.02] border border-white/[0.08] rounded-lg overflow-hidden transition-all duration-300 ${selectedId ? 'w-[50%]' : 'w-full'}`}>
          <div className="p-4 border-b border-white/[0.08] flex items-center justify-between bg-black/50">
            <div className="flex gap-3 items-center">
              <button className="px-3 py-1.5 bg-white/[0.05] border border-white/10 rounded text-sm text-white/70 hover:text-white flex items-center gap-2">
                <Filter className="w-4 h-4" /> Filter
              </button>
              {selectedRows.size > 0 && (
                <div className="flex items-center gap-2 px-3 border-l border-white/10">
                  <span className="text-sm text-white/50">{selectedRows.size} selected</span>
                  <button className="text-sm text-blue-400 hover:text-blue-300">Mark Investigating</button>
                  <button className="text-sm text-green-400 hover:text-green-300">Mark Resolved</button>
                </div>
              )}
            </div>
            <div className="relative w-64">
              <Search className="absolute left-3 top-2 w-4 h-4 text-white/40" />
              <input type="text" placeholder="Search anomalies..." className="w-full bg-white/[0.05] border border-white/10 rounded-md pl-9 pr-3 py-1.5 text-sm focus:outline-none focus:border-white/30" />
            </div>
          </div>
          <div className="flex-1 overflow-auto">
            <table className="w-full text-left">
              <thead className="sticky top-0 bg-[#0a0a0a] border-b border-white/[0.08]">
                <tr>
                  <th className="px-4 py-3 w-10">
                    <input type="checkbox" checked={selectedRows.size === anomalies.length} onChange={e => toggleAll(e.target.checked)} className="accent-blue-500 cursor-pointer" />
                  </th>
                  {['Severity', 'Service / Operation', 'Type', 'Delta', 'Status'].map(col => (
                    <th key={col} className="px-4 py-3 text-[11px] uppercase tracking-wider text-white/40 font-medium">{col}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {anomalies.map(anomaly => (
                  <tr 
                    key={anomaly.id} 
                    onClick={() => setSelectedId(selectedId === anomaly.id ? null : anomaly.id)}
                    className={`hover:bg-white/[0.04] cursor-pointer transition-colors ${selectedId === anomaly.id ? 'bg-white/[0.06]' : ''}`}
                  >
                    <td className="px-4 py-4" onClick={e => e.stopPropagation()}>
                      <input type="checkbox" checked={selectedRows.has(anomaly.id)} onChange={e => toggleRow(anomaly.id, e.target.checked)} className="accent-blue-500 cursor-pointer" />
                    </td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs uppercase tracking-wider border font-bold ${getSeverityColor(anomaly.severity)}`}>
                        {getSeverityIcon(anomaly.severity)}
                        {anomaly.severity}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium text-white/90">{anomaly.service}</span>
                        <span className="text-xs text-white/50">{anomaly.operation}</span>
                      </div>
                    </td>
                    <td className="px-4 py-4 text-sm text-white/80">{anomaly.type.replace('_', ' ')}</td>
                    <td className="px-4 py-4 font-dm-mono text-sm text-white/70">{anomaly.delta}</td>
                    <td className="px-4 py-4">
                      <div className="flex flex-col gap-1">
                        <span className={`text-sm capitalize font-medium flex items-center gap-1.5 ${getStatusColor(anomaly.status)}`}>
                          {anomaly.status === 'resolved' && <CheckCircle2 className="w-4 h-4" />}
                          {anomaly.status === 'investigating' && <Search className="w-4 h-4" />}
                          {anomaly.status === 'open' && <AlertCircle className="w-4 h-4" />}
                          {anomaly.status}
                        </span>
                        <span className="text-[11px] text-white/40">{Math.floor((Date.now() - anomaly.detectedTime)/60000)}m ago</span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* DETAILS PANEL */}
        {selectedAnomaly && (
          <div className="w-[50%] bg-white/[0.02] border border-white/[0.08] rounded-lg overflow-y-auto flex flex-col animate-in slide-in-from-right-8 duration-300">
            <div className="p-6 border-b border-white/[0.08] flex justify-between items-start sticky top-0 bg-[#0c0c0c] z-10">
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-3">
                  <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs uppercase tracking-wider border font-bold ${getSeverityColor(selectedAnomaly.severity)}`}>
                    {getSeverityIcon(selectedAnomaly.severity)} {selectedAnomaly.severity}
                  </span>
                  <span className="font-dm-mono text-white/50 text-sm">{selectedAnomaly.id}</span>
                </div>
                <h2 className="text-2xl font-bold mt-1">{selectedAnomaly.service}</h2>
                <span className="text-white/60">{selectedAnomaly.operation} • {selectedAnomaly.type.replace('_', ' ')}</span>
              </div>
              <div className="flex gap-2">
                {selectedAnomaly.status !== 'investigating' && (
                  <button onClick={() => updateStatus('investigating')} className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-sm rounded transition-colors">
                    Investigate
                  </button>
                )}
                {selectedAnomaly.status !== 'resolved' && (
                  <button onClick={() => updateStatus('resolved')} className="px-4 py-2 bg-green-500/20 hover:bg-green-500/30 text-green-400 border border-green-500/30 text-sm rounded transition-colors">
                    Resolve
                  </button>
                )}
              </div>
            </div>

            <div className="p-6 flex flex-col gap-8">
              {/* CHART */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm font-semibold uppercase tracking-wider text-white/50">Telemetry Context</h3>
                <div className="h-48 w-full bg-black/50 border border-white/[0.05] rounded p-2">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={chartData}>
                      <XAxis dataKey="time" stroke="rgba(255,255,255,0.2)" fontSize={10} />
                      <YAxis stroke="rgba(255,255,255,0.2)" fontSize={10} width={30} />
                      <Tooltip contentStyle={{ backgroundColor: '#111', border: '1px solid rgba(255,255,255,0.1)' }} />
                      <ReferenceArea x1="-20m" x2="-10m" fill="rgba(239, 68, 68, 0.15)" strokeOpacity={0.5} />
                      <Line type="monotone" dataKey="value" stroke="#3b82f6" strokeWidth={2} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* AI HYPOTHESIS */}
              <div className="flex flex-col gap-3">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-semibold uppercase tracking-wider text-white/50">Root Cause Hypothesis</h3>
                  <span className="px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 text-[10px] uppercase font-bold">AI Generated</span>
                </div>
                <div className="p-4 rounded bg-white/[0.03] border border-white/[0.05] text-white/80 leading-relaxed text-sm">
                  {selectedAnomaly.hypothesis}
                </div>
              </div>

              {/* TIMELINE */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm font-semibold uppercase tracking-wider text-white/50">Event Timeline</h3>
                <div className="relative pl-4 border-l border-white/10 ml-2 flex flex-col gap-4">
                  <div className="relative">
                    <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-blue-500" />
                    <span className="text-xs font-dm-mono text-blue-400 block mb-1">10:42 AM</span>
                    <span className="text-sm text-white/90">Deployment <span className="font-dm-mono bg-white/10 px-1 rounded">v2.4.1</span> rolled out to checkout-api</span>
                  </div>
                  <div className="relative">
                    <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-red-500" />
                    <span className="text-xs font-dm-mono text-red-400 block mb-1">10:45 AM</span>
                    <span className="text-sm text-white/90">Anomaly detected: {selectedAnomaly.delta}</span>
                  </div>
                  <div className="relative">
                    <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-amber-500" />
                    <span className="text-xs font-dm-mono text-amber-400 block mb-1">10:46 AM</span>
                    <span className="text-sm text-white/90">Alert paged to on-call engineer (PagerDuty)</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
