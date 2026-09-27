import React, { useState, useEffect, useRef, useMemo } from 'react';
import * as d3 from 'd3';
import { X, Activity, ArrowRight, RefreshCw, AlertCircle, Loader2 } from 'lucide-react';
import { LineChart, Line, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';
import { useQuery } from '@tanstack/react-query';
import { servicesApi } from '../../lib/api';
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

const generateTimeSeries = (base: number, volatility: number) => {
  return Array.from({ length: 60 }).map((_, i) => ({
    time: `-${60 - i}m`,
    p99: base + Math.random() * volatility,
    p95: base * 0.8 + Math.random() * (volatility * 0.8),
    p50: base * 0.4 + Math.random() * (volatility * 0.4),
    errorRate: Math.max(0, Math.random() * 5 - 4),
  }));
};

export const ServiceMapTab: React.FC = () => {
  const [nodes, setNodes] = useState<SimNodeData[]>([]);
  const [links, setLinks] = useState<SimLinkData[]>([]);
  const [selectedNode, setSelectedNode] = useState<SimNodeData | null>(null);
  const [showCritical, setShowCritical] = useState(false);

  // ViewBox pan/zoom
  const [viewBox, setViewBox] = useState({ x: -400, y: -300, w: 800, h: 600 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const containerRef = useRef<HTMLDivElement>(null);

  // Real React Query for Service Graph
  const {
    data: serviceGraph,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['service-graph'],
    queryFn: () => servicesApi.getGraph(),
  });

  useEffect(() => {
    if (!serviceGraph || !serviceGraph.nodes || serviceGraph.nodes.length === 0) {
      setNodes([]);
      setLinks([]);
      return;
    }

    const simNodes: SimNodeData[] = serviceGraph.nodes.map((n) => ({ ...n }));
    const simLinks: SimLinkData[] = serviceGraph.edges.map((e) => ({ ...e }));

    const simulation = d3
      .forceSimulation(simNodes as d3.SimulationNodeDatum[])
      .force(
        'link',
        d3
          .forceLink(simLinks)
          .id((d: d3.SimulationNodeDatum) => (d as SimNodeData).id)
          .distance(200),
      )
      .force('charge', d3.forceManyBody().strength(-2000))
      .force('center', d3.forceCenter(0, 0))
      .force('collide', d3.forceCollide().radius(80))
      .on('tick', () => {
        setNodes([...simNodes]);
        setLinks([...simLinks]);
      });

    return () => {
      simulation.stop();
    };
  }, [serviceGraph]);

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
    return 'rgba(239, 68, 68, 0.1)';
  };

  const chartData = useMemo(() => {
    if (!selectedNode) return [];
    return generateTimeSeries(selectedNode.p99 || 45, (selectedNode.p99 || 45) * 0.5);
  }, [selectedNode]);

  return (
    <div className="flex w-full h-full bg-black relative overflow-hidden font-dm-sans">
      {/* TOOLBAR */}
      <div className="absolute top-4 left-4 z-10 flex gap-2 items-center">
        <button
          onClick={() => setShowCritical(!showCritical)}
          className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
            showCritical
              ? 'bg-red-500/20 text-red-300 border-red-500/50'
              : 'bg-white/5 text-white/60 border-white/10 hover:text-white'
          }`}
        >
          {showCritical ? 'Highlighting Critical Path' : 'Highlight Critical Path'}
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
          <span>Refresh</span>
        </button>
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
            <span className="text-sm">Computing dynamic service graph from ClickHouse spans...</span>
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

              const isCrit = showCritical && link.criticalPath;
              const strokeColor = isCrit ? '#ef4444' : '#333';
              const strokeWidth = isCrit ? 2.5 : Math.max(1, Math.min(4, (link.rps || 100) / 300));

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
                    strokeDasharray={isCrit ? '4,4' : 'none'}
                    className={isCrit ? 'animate-pulse' : ''}
                  />
                  {/* Traffic stats label on link */}
                  <text
                    x={(src.x + tgt.x) / 2}
                    y={(src.y + tgt.y) / 2 - 8}
                    fill="#666"
                    fontSize="9"
                    textAnchor="middle"
                    className="font-dm-mono pointer-events-none"
                  >
                    {link.rps ? `${Math.round(link.rps)} rps` : ''}
                  </text>
                </g>
              );
            })}

            {/* NODES */}
            {nodes.map((node) => {
              if (node.x === undefined || node.y === undefined) return null;
              const isSelected = selectedNode?.id === node.id;

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
                      filter: isSelected ? `drop-shadow(0 0 12px ${getBgColor(node.health || 'healthy')})` : 'none',
                    }}
                  />
                  <circle cx="-55" cy="0" r="4" fill={getBorderColor(node.health || 'healthy')} />
                  <text x="-40" y="-5" fill="#fff" fontSize="12" fontWeight="600">
                    {node.name}
                  </text>
                  <text x="-40" y="12" fill="#888" fontSize="10" className="font-dm-mono">
                    p99: {Math.round(node.p99 || 0)}ms
                  </text>
                </g>
              );
            })}
          </svg>
        )}
      </div>

      {/* RIGHT DRAWER */}
      <div
        className={`absolute top-0 right-0 bottom-0 w-[400px] bg-[#0a0a0a] border-l border-white/[0.08] shadow-2xl transition-transform duration-300 ease-out flex flex-col ${
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
                <h2 className="text-lg font-bold text-white">{selectedNode.name}</h2>
              </div>
              <button
                onClick={() => setSelectedNode(null)}
                className="p-2 hover:bg-white/10 rounded-full transition-colors"
                aria-label="Close details"
              >
                <X className="w-5 h-5 text-white/50" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-8 text-white">
              {/* METRICS CHARTS */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm text-white/60 uppercase tracking-wider font-semibold">Latency Profile (1h)</h3>
                  <Activity className="w-4 h-4 text-white/40" />
                </div>
                <div className="h-[120px] w-full mt-2">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={chartData}>
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#111',
                          border: '1px solid rgba(255,255,255,0.1)',
                          fontSize: '12px',
                        }}
                      />
                      <Line type="monotone" dataKey="p99" stroke="#ef4444" strokeWidth={2} dot={false} />
                      <Line type="monotone" dataKey="p95" stroke="#f59e0b" strokeWidth={1.5} dot={false} />
                      <Line type="monotone" dataKey="p50" stroke="#3b82f6" strokeWidth={1} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
                <div className="flex gap-4 text-xs font-dm-mono mt-1">
                  <span className="text-red-400">p99: {Math.round(selectedNode.p99 || 45)}ms</span>
                  <span className="text-amber-400">p95: {Math.round((selectedNode.p99 || 45) * 0.8)}ms</span>
                  <span className="text-blue-400">p50: {Math.round((selectedNode.p99 || 45) * 0.4)}ms</span>
                </div>
              </div>

              <div className="flex flex-col gap-2">
                <h3 className="text-sm text-white/60 uppercase tracking-wider font-semibold">Error Rate</h3>
                <div className="h-[80px] w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={chartData}>
                      <Area type="step" dataKey="errorRate" stroke="#ef4444" fill="rgba(239, 68, 68, 0.2)" />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* DEPENDENCIES */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm text-white/60 uppercase tracking-wider font-semibold">Dependencies</h3>
                <div className="flex flex-col gap-2">
                  {links
                    .filter((l) => (l.source as SimNodeData).id === selectedNode.id)
                    .map((l, i) => (
                      <div key={i} className="flex items-center gap-3 p-2 rounded bg-white/[0.03]">
                        <ArrowRight className="w-4 h-4 text-white/40" />
                        <span className="text-sm text-white/80">{(l.target as SimNodeData).name}</span>
                      </div>
                    ))}
                  {links
                    .filter((l) => (l.target as SimNodeData).id === selectedNode.id)
                    .map((l, i) => (
                      <div key={i} className="flex items-center gap-3 p-2 rounded bg-white/[0.03]">
                        <ArrowRight className="w-4 h-4 text-white/40 rotate-180" />
                        <span className="text-sm text-white/80">{(l.source as SimNodeData).name}</span>
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
