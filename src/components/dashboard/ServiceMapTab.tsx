import React, { useState, useEffect, useRef, useMemo } from 'react';
import * as d3 from 'd3';
import { X, Activity, AlertTriangle, ArrowRight } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';

interface NodeData {
  id: string;
  name: string;
  p99: number;
  health: 'healthy' | 'degraded' | 'critical';
  x?: number;
  y?: number;
  vx?: number;
  vy?: number;
}

interface LinkData {
  source: string | NodeData;
  target: string | NodeData;
  rps: number;
  errorRate: number;
  criticalPath?: boolean;
}

const mockNodes: NodeData[] = [
  { id: 'api-gateway', name: 'api-gateway', p99: 45, health: 'healthy' },
  { id: 'checkout-api', name: 'checkout-api', p99: 120, health: 'degraded' },
  { id: 'payment-svc', name: 'payment-svc', p99: 450, health: 'critical' },
  { id: 'inventory-svc', name: 'inventory-svc', p99: 35, health: 'healthy' },
  { id: 'user-service', name: 'user-service', p99: 20, health: 'healthy' },
  { id: 'notifications-svc', name: 'notifications-svc', p99: 15, health: 'healthy' },
];

const mockLinks: LinkData[] = [
  { source: 'api-gateway', target: 'checkout-api', rps: 1200, errorRate: 2.5, criticalPath: true },
  { source: 'api-gateway', target: 'user-service', rps: 800, errorRate: 0.1 },
  { source: 'checkout-api', target: 'payment-svc', rps: 450, errorRate: 15.2, criticalPath: true },
  { source: 'checkout-api', target: 'inventory-svc', rps: 900, errorRate: 0.5 },
  { source: 'payment-svc', target: 'notifications-svc', rps: 400, errorRate: 0.1 },
];

const generateTimeSeries = (base: number, volatility: number) => {
  return Array.from({ length: 60 }).map((_, i) => ({
    time: `-${60 - i}m`,
    p99: base + Math.random() * volatility,
    p95: base * 0.8 + Math.random() * (volatility * 0.8),
    p50: base * 0.4 + Math.random() * (volatility * 0.4),
    errorRate: Math.max(0, (Math.random() * 5) - 4) // mostly 0, some spikes
  }));
};

export const ServiceMapTab: React.FC = () => {
  const [nodes, setNodes] = useState<NodeData[]>([]);
  const [links, setLinks] = useState<LinkData[]>([]);
  const [selectedNode, setSelectedNode] = useState<NodeData | null>(null);
  const [showCritical, setShowCritical] = useState(false);
  
  // ViewBox pan/zoom
  const [viewBox, setViewBox] = useState({ x: -400, y: -300, w: 800, h: 600 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Deep copy to allow d3 to mutate without React strict mode issues
    const simNodes = mockNodes.map(n => ({ ...n }));
    const simLinks = mockLinks.map(l => ({ ...l }));

    const simulation = d3.forceSimulation(simNodes as d3.SimulationNodeDatum[])
      .force('link', d3.forceLink(simLinks).id((d: any) => d.id).distance(200))
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
  }, []);

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsDragging(true);
    setDragStart({ x: e.clientX, y: e.clientY });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging) return;
    const dx = e.clientX - dragStart.x;
    const dy = e.clientY - dragStart.y;
    setViewBox(prev => ({ ...prev, x: prev.x - dx, y: prev.y - dy }));
    setDragStart({ x: e.clientX, y: e.clientY });
  };

  const handleWheel = (e: React.WheelEvent) => {
    const zoomFactor = e.deltaY > 0 ? 1.1 : 0.9;
    setViewBox(prev => ({
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
    return generateTimeSeries(selectedNode.p99, selectedNode.p99 * 0.5);
  }, [selectedNode]);

  return (
    <div className="flex w-full h-full bg-black relative overflow-hidden font-dm-sans">
      {/* TOOLBAR */}
      <div className="absolute top-4 left-4 z-10 flex gap-2">
        <button 
          onClick={() => setShowCritical(!showCritical)}
          className={`px-4 py-2 rounded text-sm transition-colors ${showCritical ? 'bg-red-500/20 text-red-400 border border-red-500/50' : 'bg-white/[0.05] text-white/70 border border-white/10 hover:bg-white/10'}`}
        >
          {showCritical ? 'Hide Critical Path' : 'Show Critical Path'}
        </button>
      </div>

      {/* SVG CANVAS */}
      <div 
        ref={containerRef}
        className="flex-1 cursor-grab active:cursor-grabbing"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={() => setIsDragging(false)}
        onMouseLeave={() => setIsDragging(false)}
        onWheel={handleWheel}
      >
        <svg viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.w} ${viewBox.h}`} className="w-full h-full">
          <defs>
            <marker id="arrowhead" viewBox="0 0 10 10" refX="28" refY="5" markerWidth="6" markerHeight="6" orient="auto">
              <path d="M 0 0 L 10 5 L 0 10 z" fill="#ffffff" opacity="0.3" />
            </marker>
            <marker id="arrowhead-critical" viewBox="0 0 10 10" refX="28" refY="5" markerWidth="6" markerHeight="6" orient="auto">
              <path d="M 0 0 L 10 5 L 0 10 z" fill="#ef4444" />
            </marker>
          </defs>

          {/* EDGES */}
          {links.map((link, i) => {
            const source = link.source as NodeData;
            const target = link.target as NodeData;
            if (source.x === undefined || target.x === undefined) return null;
            
            const isCritical = showCritical && link.criticalPath;
            const midX = (source.x + target.x) / 2;
            const midY = (source.y + target.y) / 2;

            return (
              <g key={i}>
                <line
                  x1={source.x} y1={source.y}
                  x2={target.x} y2={target.y}
                  stroke={isCritical ? '#ef4444' : '#ffffff'}
                  strokeWidth={isCritical ? 3 : 1.5}
                  strokeOpacity={isCritical ? 1 : 0.2}
                  markerEnd={`url(#${isCritical ? 'arrowhead-critical' : 'arrowhead'})`}
                  className="transition-all duration-300"
                />
                <rect x={midX - 25} y={midY - 10} width="50" height="20" fill="#000" rx="4" opacity="0.8" />
                <text x={midX} y={midY + 4} textAnchor="middle" fill={isCritical ? '#ef4444' : '#fff'} opacity={isCritical ? 1 : 0.6} fontSize="10" className="font-dm-mono">
                  {link.rps} req/s
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
                onClick={(e) => { e.stopPropagation(); setSelectedNode(node); }}
                className="cursor-pointer group transition-transform duration-200"
              >
                <rect 
                  x="-75" y="-30" 
                  width="150" height="60" 
                  rx="8" 
                  fill="#111" 
                  stroke={isSelected ? '#fff' : getBorderColor(node.health)} 
                  strokeWidth={isSelected ? 2 : 1}
                  className="transition-all duration-200 group-hover:brightness-125"
                  style={{ filter: isSelected ? `drop-shadow(0 0 12px ${getBgColor(node.health)})` : 'none' }}
                />
                <circle cx="-55" cy="0" r="4" fill={getBorderColor(node.health)} />
                <text x="-40" y="-5" fill="#fff" fontSize="12" fontWeight="600">{node.name}</text>
                <text x="-40" y="12" fill="#888" fontSize="10" className="font-dm-mono">p99: {node.p99}ms</text>
              </g>
            );
          })}
        </svg>
      </div>

      {/* RIGHT DRAWER */}
      <div 
        className={`absolute top-0 right-0 bottom-0 w-[400px] bg-[#0a0a0a] border-l border-white/[0.08] shadow-2xl transition-transform duration-300 ease-out flex flex-col ${selectedNode ? 'translate-x-0' : 'translate-x-full'}`}
      >
        {selectedNode && (
          <>
            <div className="p-6 border-b border-white/[0.08] flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full" style={{ backgroundColor: getBorderColor(selectedNode.health) }} />
                <h2 className="text-lg font-bold">{selectedNode.name}</h2>
              </div>
              <button onClick={() => setSelectedNode(null)} className="p-2 hover:bg-white/10 rounded-full transition-colors">
                <X className="w-5 h-5 text-white/50" />
              </button>
            </div>
            
            <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-8">
              {/* METRICS CHARTS */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm text-white/60 uppercase tracking-wider font-semibold">Latency Profile (1h)</h3>
                  <Activity className="w-4 h-4 text-white/40" />
                </div>
                <div className="h-[120px] w-full mt-2">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={chartData}>
                      <Tooltip contentStyle={{ backgroundColor: '#111', border: '1px solid rgba(255,255,255,0.1)', fontSize: '12px' }} />
                      <Line type="monotone" dataKey="p99" stroke="#ef4444" strokeWidth={2} dot={false} />
                      <Line type="monotone" dataKey="p95" stroke="#f59e0b" strokeWidth={1.5} dot={false} />
                      <Line type="monotone" dataKey="p50" stroke="#3b82f6" strokeWidth={1} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
                <div className="flex gap-4 text-xs font-dm-mono mt-1">
                  <span className="text-red-400">p99: {Math.round(selectedNode.p99)}ms</span>
                  <span className="text-amber-400">p95: {Math.round(selectedNode.p99 * 0.8)}ms</span>
                  <span className="text-blue-400">p50: {Math.round(selectedNode.p99 * 0.4)}ms</span>
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

              {/* SLOWEST OPERATIONS */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm text-white/60 uppercase tracking-wider font-semibold">Slowest Operations</h3>
                <div className="flex flex-col gap-2">
                  {['process_batch', 'verify_token', 'fetch_relations', 'update_metrics', 'sync_db'].map((op, i) => (
                    <div key={op} className="flex items-center justify-between p-2 rounded bg-white/[0.03] border border-white/[0.05]">
                      <span className="text-sm text-white/80">{op}</span>
                      <span className="font-dm-mono text-xs text-amber-400">{Math.round(selectedNode.p99 * (1 - i*0.1))}ms</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* DEPENDENCIES */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm text-white/60 uppercase tracking-wider font-semibold">Dependencies</h3>
                <div className="flex flex-col gap-2">
                  {mockLinks.filter(l => (l.source as NodeData).id === selectedNode.id).map((l, i) => (
                    <div key={i} className="flex items-center gap-3 p-2 rounded bg-white/[0.03]">
                      <ArrowRight className="w-4 h-4 text-white/40" />
                      <span className="text-sm text-white/80">{(l.target as NodeData).name}</span>
                    </div>
                  ))}
                  {mockLinks.filter(l => (l.target as NodeData).id === selectedNode.id).map((l, i) => (
                    <div key={i} className="flex items-center gap-3 p-2 rounded bg-white/[0.03]">
                      <ArrowRight className="w-4 h-4 text-white/40 rotate-180" />
                      <span className="text-sm text-white/80">{(l.source as NodeData).name}</span>
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
