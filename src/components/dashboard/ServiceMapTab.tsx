import React, { useState, useEffect, useRef, useMemo } from 'react';
import * as d3 from 'd3';
import { X, Activity, ArrowRight, RefreshCw, AlertCircle, Loader2, AlertTriangle, ShieldCheck, Zap } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { servicesApi, anomaliesApi } from '../../lib/api';
import type { ServiceNode, ServiceEdge } from '../../lib/api';

export interface SimNodeData extends ServiceNode {
  x?: number;
  y?: number;
  vx?: number;
  vy?: number;
}

export interface SimLinkData extends Omit<ServiceEdge, 'source' | 'target'> {
  source: string | SimNodeData;
  target: string | SimNodeData;
}

export const ServiceMapTab: React.FC = () => {
  const [nodes, setNodes] = useState<SimNodeData[]>([]);
  const [links, setLinks] = useState<SimLinkData[]>([]);
  const [selectedNode, setSelectedNode] = useState<SimNodeData | null>(null);
  const [showCriticalOnly, setShowCriticalOnly] = useState(false);
  const [highlightCriticalPath, setHighlightCriticalPath] = useState(true);

  // ViewBox pan/zoom
  const [viewBox, setViewBox] = useState({ x: -400, y: -300, w: 800, h: 600 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const containerRef = useRef<HTMLDivElement>(null);

  // 1. Real-time Service Graph Query with 10s Adaptive Polling (Decision #29)
  const {
    data: serviceGraph,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
    dataUpdatedAt,
  } = useQuery({
    queryKey: ['service-graph'],
    queryFn: () => servicesApi.getGraph(),
    refetchInterval: 10000, // 10s auto-refresh interval when active
    refetchOnWindowFocus: true,
  });

  // 2. Real Service Stats Query for Selected Node
  const { data: selectedStats, isLoading: statsLoading } = useQuery({
    queryKey: ['service-stats', selectedNode?.name],
    queryFn: () => servicesApi.getStats(selectedNode!.name),
    enabled: !!selectedNode,
    refetchInterval: 10000,
  });

  // 3. Real Active Anomalies for Selected Node (Phase 3 Integration)
  const { data: activeAnomalies, isLoading: anomaliesLoading } = useQuery({
    queryKey: ['service-anomalies', selectedNode?.name],
    queryFn: () => anomaliesApi.list({ service: selectedNode!.name, status: 'open' }),
    enabled: !!selectedNode,
    refetchInterval: 10000,
  });

  // D3 Force Simulation Setup
  useEffect(() => {
    if (!serviceGraph || !serviceGraph.nodes || serviceGraph.nodes.length === 0) {
      setNodes([]);
      setLinks([]);
      return;
    }

    const filteredNodes = showCriticalOnly
      ? serviceGraph.nodes.filter((n) => n.health !== 'healthy')
      : serviceGraph.nodes;

    const nodeIds = new Set(filteredNodes.map((n) => n.id));
    const filteredEdges = serviceGraph.edges.filter(
      (e) => nodeIds.has(e.source) && nodeIds.has(e.target),
    );

    const simNodes: SimNodeData[] = filteredNodes.map((n) => ({ ...n }));
    const simLinks: SimLinkData[] = filteredEdges.map((e) => ({ ...e }));

    const simulation = d3
      .forceSimulation(simNodes as d3.SimulationNodeDatum[])
      .force(
        'link',
        d3
          .forceLink(simLinks)
          .id((d: d3.SimulationNodeDatum) => (d as SimNodeData).id)
          .distance(180),
      )
      .force('charge', d3.forceManyBody().strength(-1800))
      .force('center', d3.forceCenter(0, 0))
      .force('collide', d3.forceCollide().radius(90))
      .on('tick', () => {
        setNodes([...simNodes]);
        setLinks([...simLinks]);
      });

    return () => {
      simulation.stop();
    };
  }, [serviceGraph, showCriticalOnly]);

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsDragging(true);
    setDragStart({ x: e.clientX, y: e.clientY });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging) return;
    const dx = e.clientX - dragStart.x;
    const dy = e.clientY - dragStart.y;
    setViewBox((prev) => ({ ...prev, x: prev.x - dx, y: prev.y - dy }));
    setDragStart({ x: e.clientX, y: e.clientY });
  };

  const handleWheel = (e: React.WheelEvent) => {
    const zoomFactor = e.deltaY > 0 ? 1.1 : 0.9;
    setViewBox((prev) => ({
      ...prev,
      w: prev.w * zoomFactor,
      h: prev.h * zoomFactor,
      x: prev.x - (prev.w * zoomFactor - prev.w) / 2,
      y: prev.y - (prev.h * zoomFactor - prev.h) / 2,
    }));
  };

  const getBorderColor = (health: string) => {
    if (health === 'healthy') return '#22c55e';
    if (health === 'degraded') return '#f59e0b';
    return '#ef4444';
  };

  const getBgColor = (health: string) => {
    if (health === 'healthy') return 'rgba(34, 197, 94, 0.1)';
    if (health === 'degraded') return 'rgba(245, 158, 11, 0.1)';
    return 'rgba(239, 68, 68, 0.15)';
  };

  const totalDegradedOrCritical = useMemo(() => {
    if (!serviceGraph?.nodes) return 0;
    return serviceGraph.nodes.filter((n) => n.health !== 'healthy').length;
  }, [serviceGraph]);

  return (
    <div className="flex w-full h-full bg-black relative overflow-hidden font-dm-sans">
      {/* TOOLBAR */}
      <div className="absolute top-4 left-4 z-10 flex flex-wrap gap-2 items-center">
        <button
          onClick={() => setHighlightCriticalPath(!highlightCriticalPath)}
          className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors flex items-center gap-1.5 ${
            highlightCriticalPath
              ? 'bg-amber-500/20 text-amber-300 border-amber-500/50'
              : 'bg-white/5 text-white/60 border-white/10 hover:text-white'
          }`}
        >
          <Zap className="w-3.5 h-3.5" />
          <span>{highlightCriticalPath ? 'Critical Path Highlighted' : 'Highlight Critical Path'}</span>
        </button>

        <button
          onClick={() => setShowCriticalOnly(!showCriticalOnly)}
          className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors flex items-center gap-1.5 ${
            showCriticalOnly
              ? 'bg-red-500/20 text-red-300 border-red-500/50'
              : 'bg-white/5 text-white/60 border-white/10 hover:text-white'
          }`}
        >
          <AlertTriangle className="w-3.5 h-3.5" />
          <span>
            {showCriticalOnly ? 'Showing Anomalous Services' : `Filter Degraded (${totalDegradedOrCritical})`}
          </span>
        </button>

        <button
          onClick={() => setViewBox({ x: -400, y: -300, w: 800, h: 600 })}
          className="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/5 text-white/60 border border-white/10 hover:text-white transition-colors"
        >
          Reset Zoom
        </button>

        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-white/5 text-white/60 border border-white/10 hover:text-white transition-colors"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          <span>{isFetching ? 'Updating...' : 'Refresh'}</span>
        </button>

        {dataUpdatedAt > 0 && (
          <span className="text-[11px] text-white/40 ml-2 font-dm-mono">
            Auto-sync: 10s • Updated {new Date(dataUpdatedAt).toLocaleTimeString()}
          </span>
        )}
      </div>

      {/* ERROR BANNER */}
      {isError && (
        <div className="absolute top-16 left-4 right-4 z-10 p-3 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-between text-xs">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-amber-400 shrink-0" />
            <span className="text-white/80">
              Backend service unreachable:{' '}
              <span className="text-amber-300">{(error as Error)?.message || 'http://localhost:8080'}</span>
            </span>
          </div>
          <button
            onClick={() => refetch()}
            className="text-amber-300 hover:underline font-medium"
          >
            Retry
          </button>
        </div>
      )}

      {/* GRAPH CANVAS */}
      <div
        ref={containerRef}
        className="w-full h-full cursor-grab active:cursor-grabbing"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={() => setIsDragging(false)}
        onWheel={handleWheel}
      >
        {isLoading ? (
          <div className="flex flex-col items-center justify-center h-full text-white/50 gap-2">
            <Loader2 className="w-6 h-6 animate-spin text-blue-400" />
            <span className="text-sm">Computing service topology graph & anomaly health from ClickHouse...</span>
          </div>
        ) : nodes.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-white/40 gap-3">
            <Activity className="w-10 h-10 opacity-30 text-blue-400" />
            <div className="text-center max-w-md">
              <p className="text-sm font-semibold text-white/80">No active service topology detected</p>
              <p className="text-xs text-white/50 mt-1">
                Ingest distributed traces with parent-child span relationships to build the service dependency topology graph in real time.
              </p>
            </div>
          </div>
        ) : (
          <svg
            className="w-full h-full select-none"
            viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.w} ${viewBox.h}`}
          >
            <defs>
              <marker id="arrow" viewBox="0 -5 10 10" refX="28" refY="0" markerWidth="6" markerHeight="6" orient="auto">
                <path d="M0,-5L10,0L0,5" fill="#444" />
              </marker>
              <marker id="arrow-critical" viewBox="0 -5 10 10" refX="28" refY="0" markerWidth="6" markerHeight="6" orient="auto">
                <path d="M0,-5L10,0L0,5" fill="#ef4444" />
              </marker>
            </defs>

            {/* LINKS */}
            {links.map((link, i) => {
              const src = link.source as SimNodeData;
              const tgt = link.target as SimNodeData;
              if (src.x === undefined || src.y === undefined || tgt.x === undefined || tgt.y === undefined) return null;

              const isCrit = highlightCriticalPath && link.criticalPath;
              const strokeColor = isCrit ? '#ef4444' : '#333';
              const strokeWidth = isCrit ? 2.5 : Math.max(1, Math.min(4, (link.rps || 10) / 50));

              return (
                <g key={i} className="transition-all duration-300">
                  <line
                    x1={src.x}
                    y1={src.y}
                    x2={tgt.x}
                    y2={tgt.y}
                    stroke={strokeColor}
                    strokeWidth={strokeWidth}
                    markerEnd={isCrit ? 'url(#arrow-critical)' : 'url(#arrow)'}
                    strokeDasharray={isCrit ? '5,5' : 'none'}
                    className={isCrit ? 'animate-pulse' : ''}
                  />
                  {/* Traffic stats label on link */}
                  <text
                    x={(src.x + tgt.x) / 2}
                    y={(src.y + tgt.y) / 2 - 8}
                    fill={isCrit ? '#ef4444' : '#777'}
                    fontSize="9"
                    textAnchor="middle"
                    className="font-dm-mono pointer-events-none"
                  >
                    {link.p95Ms && link.p95Ms > 0 ? `${Math.round(link.p95Ms)}ms | ` : ''}
                    {link.rps ? `${Math.round(link.rps)} rps` : ''}
                  </text>
                </g>
              );
            })}

            {/* NODES */}
            {nodes.map((node) => {
              if (node.x === undefined || node.y === undefined) return null;
              const isSelected = selectedNode?.id === node.id;
              const hasAnomalies = (node.activeAnomalies || 0) > 0;

              return (
                <g
                  key={node.id}
                  transform={`translate(${node.x},${node.y})`}
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedNode(node);
                  }}
                  className="cursor-pointer group"
                >
                  <rect
                    x="-75"
                    y="-30"
                    width="150"
                    height="60"
                    rx="8"
                    fill="#111"
                    stroke={isSelected ? '#fff' : getBorderColor(node.health || 'healthy')}
                    strokeWidth={isSelected ? 2 : 1}
                    className="transition-all duration-200 group-hover:brightness-125"
                    style={{
                      filter: isSelected
                        ? `drop-shadow(0 0 12px ${getBgColor(node.health || 'healthy')})`
                        : hasAnomalies
                        ? 'drop-shadow(0 0 8px rgba(239, 68, 68, 0.4))'
                        : 'none',
                    }}
                  />
                  <circle cx="-55" cy="0" r="4" fill={getBorderColor(node.health || 'healthy')} />
                  <text x="-40" y="-5" fill="#fff" fontSize="12" fontWeight="600">
                    {node.name}
                  </text>
                  <text x="-40" y="12" fill="#888" fontSize="10" className="font-dm-mono">
                    p99: {Math.round(node.p99 || 0)}ms
                  </text>

                  {/* ACTIVE ANOMALY BADGE */}
                  {hasAnomalies && (
                    <g transform="translate(65, -25)">
                      <circle r="9" fill="#ef4444" className="animate-pulse" />
                      <text fill="#fff" fontSize="9" fontWeight="bold" textAnchor="middle" dy="3.5">
                        {node.activeAnomalies}
                      </text>
                    </g>
                  )}
                </g>
              );
            })}
          </svg>
        )}
      </div>

      {/* RIGHT DRAWER */}
      <div
        className={`absolute top-0 right-0 bottom-0 w-[420px] bg-[#0a0a0a] border-l border-white/[0.08] shadow-2xl transition-transform duration-300 ease-out flex flex-col z-20 ${
          selectedNode ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        {selectedNode && (
          <>
            <div className="p-6 border-b border-white/[0.08] flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: getBorderColor(selectedNode.health || 'healthy') }}
                />
                <div>
                  <h2 className="text-lg font-bold text-white leading-tight">{selectedNode.name}</h2>
                  <span className="text-xs uppercase font-medium tracking-wider" style={{ color: getBorderColor(selectedNode.health || 'healthy') }}>
                    Status: {selectedNode.health || 'healthy'}
                  </span>
                </div>
              </div>
              <button
                onClick={() => setSelectedNode(null)}
                className="p-2 hover:bg-white/10 rounded-full transition-colors"
                aria-label="Close details"
              >
                <X className="w-5 h-5 text-white/50" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-6 text-white">
              {/* LATENCY METRICS (REAL DATA) */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <h3 className="text-xs text-white/60 uppercase tracking-wider font-semibold">Latency Percentiles (1h)</h3>
                  {statsLoading && <Loader2 className="w-3.5 h-3.5 animate-spin text-white/40" />}
                </div>

                <div className="grid grid-cols-3 gap-2">
                  <div className="p-3 rounded-lg bg-white/[0.03] border border-white/[0.06]">
                    <span className="text-[10px] text-white/40 uppercase">p50</span>
                    <p className="text-base font-bold text-blue-400 font-dm-mono mt-0.5">
                      {Math.round(selectedStats?.p50 || selectedNode.p99 * 0.4 || 0)}ms
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-white/[0.03] border border-white/[0.06]">
                    <span className="text-[10px] text-white/40 uppercase">p95</span>
                    <p className="text-base font-bold text-amber-400 font-dm-mono mt-0.5">
                      {Math.round(selectedStats?.p95 || selectedNode.p99 * 0.8 || 0)}ms
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-white/[0.03] border border-white/[0.06]">
                    <span className="text-[10px] text-white/40 uppercase">p99</span>
                    <p className="text-base font-bold text-red-400 font-dm-mono mt-0.5">
                      {Math.round(selectedStats?.p99 || selectedNode.p99 || 0)}ms
                    </p>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-2 mt-1">
                  <div className="p-3 rounded-lg bg-white/[0.03] border border-white/[0.06]">
                    <span className="text-[10px] text-white/40 uppercase">Request Rate</span>
                    <p className="text-sm font-semibold text-white/90 font-dm-mono mt-0.5">
                      {selectedStats?.request_rate ? `${selectedStats.request_rate.toFixed(1)} rps` : 'Live'}
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-white/[0.03] border border-white/[0.06]">
                    <span className="text-[10px] text-white/40 uppercase">Error Rate</span>
                    <p className={`text-sm font-semibold font-dm-mono mt-0.5 ${
                      (selectedStats?.error_rate || 0) > 1 ? 'text-red-400' : 'text-emerald-400'
                    }`}>
                      {selectedStats?.error_rate ? `${selectedStats.error_rate.toFixed(2)}%` : '0.00%'}
                    </p>
                  </div>
                </div>
              </div>

              {/* ACTIVE ANOMALIES (PHASE 3 INTEGRATION) */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <h3 className="text-xs text-white/60 uppercase tracking-wider font-semibold">
                    Active Anomalies ({activeAnomalies?.length || selectedNode.activeAnomalies || 0})
                  </h3>
                  {anomaliesLoading && <Loader2 className="w-3.5 h-3.5 animate-spin text-white/40" />}
                </div>

                {activeAnomalies && activeAnomalies.length > 0 ? (
                  <div className="flex flex-col gap-2">
                    {activeAnomalies.map((anm) => (
                      <div
                        key={anm.id}
                        className="p-3 rounded-lg bg-red-500/[0.08] border border-red-500/20 flex flex-col gap-1.5"
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-semibold text-red-400 uppercase tracking-wider">
                            {anm.severity} • {anm.operation_name}
                          </span>
                          <span className="text-[10px] text-white/40 font-dm-mono">
                            Z = {anm.z_score.toFixed(1)}
                          </span>
                        </div>
                        <div className="text-xs text-white/80">
                          Observed: <span className="font-dm-mono text-red-300 font-bold">{Math.round(anm.observed_latency_ms)}ms</span>{' '}
                          (Baseline: {Math.round(anm.baseline_latency_ms)}ms)
                        </div>
                        <div className="text-[11px] text-white/50 flex items-center gap-1">
                          <span>Root Cause:</span>
                          <span className="text-amber-300 font-medium">{anm.root_cause_service}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="p-3 rounded-lg bg-emerald-500/[0.05] border border-emerald-500/20 flex items-center gap-2.5">
                    <ShieldCheck className="w-4 h-4 text-emerald-400 shrink-0" />
                    <span className="text-xs text-emerald-300">
                      Zero unresolved anomalies detected. Latency and error baselines nominal.
                    </span>
                  </div>
                )}
              </div>

              {/* DEPENDENCIES */}
              <div className="flex flex-col gap-3">
                <h3 className="text-xs text-white/60 uppercase tracking-wider font-semibold">Service Dependencies</h3>
                <div className="flex flex-col gap-2">
                  {links
                    .filter((l) => (l.source as SimNodeData).id === selectedNode.id)
                    .map((l, i) => (
                      <div key={i} className="flex items-center justify-between p-2 rounded bg-white/[0.03] border border-white/[0.05]">
                        <div className="flex items-center gap-2.5">
                          <ArrowRight className="w-4 h-4 text-white/40" />
                          <span className="text-xs text-white/90">{(l.target as SimNodeData).name}</span>
                        </div>
                        <span className="text-[11px] text-white/50 font-dm-mono">
                          {l.p95Ms ? `${Math.round(l.p95Ms)}ms • ` : ''}
                          {Math.round(l.rps || 0)} rps
                        </span>
                      </div>
                    ))}
                  {links
                    .filter((l) => (l.target as SimNodeData).id === selectedNode.id)
                    .map((l, i) => (
                      <div key={i} className="flex items-center justify-between p-2 rounded bg-white/[0.03] border border-white/[0.05]">
                        <div className="flex items-center gap-2.5">
                          <ArrowRight className="w-4 h-4 text-white/40 rotate-180" />
                          <span className="text-xs text-white/90">{(l.source as SimNodeData).name}</span>
                        </div>
                        <span className="text-[11px] text-white/50 font-dm-mono">
                          {l.p95Ms ? `${Math.round(l.p95Ms)}ms • ` : ''}
                          {Math.round(l.rps || 0)} rps
                        </span>
                      </div>
                    ))}
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
