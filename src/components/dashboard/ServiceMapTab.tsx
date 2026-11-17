import React, { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  User,
  Layers,
  Box,
  Server,
  CreditCard,
  Database,
  Bell,
  Maximize2,
  RefreshCw,
  Zap,
} from 'lucide-react';
import { servicesApi } from '../../lib/api';
import { Sparkline } from '../common/Sparkline';
import { Select } from '../common/Select';
import {
  DEMO_TOPOLOGY_NODES,
  DEMO_TOPOLOGY_EDGES,
  type TopologyNode,
} from '../../lib/fixtures';

export const ServiceMapTab: React.FC = () => {
  const [timeRange, setTimeRange] = useState('1h');
  const [selectedNodeId, setSelectedNodeId] = useState<string>('checkout-api');

  // Real backend query for services & graph
  const {
    data: liveGraph,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['service-graph'],
    queryFn: () => servicesApi.getGraph(),
  });

  // Use demo nodes as baseline coordinates and blend with live graph if present
  const nodes: TopologyNode[] = useMemo(() => {
    if (!liveGraph?.nodes || liveGraph.nodes.length === 0) {
      return DEMO_TOPOLOGY_NODES;
    }
    // Blend live nodes with coordinates
    return DEMO_TOPOLOGY_NODES.map((dNode) => {
      const match = liveGraph.nodes.find((ln) => ln.name === dNode.name);
      if (match) {
        return {
          ...dNode,
          p99: match.p99 || dNode.p99,
          health: match.health || dNode.health,
        };
      }
      return dNode;
    });
  }, [liveGraph]);

  const edges = DEMO_TOPOLOGY_EDGES;

  // Selected node details
  const selectedNode = useMemo(() => {
    return nodes.find((n) => n.id === selectedNodeId) || nodes[3]; // default checkout-api
  }, [nodes, selectedNodeId]);

  // Node Icon Helper
  const getNodeIcon = (type: string, id: string) => {
    if (type === 'client') return <User className="w-4 h-4 text-[#9aa0b5]" />;
    if (type === 'gateway') return <Layers className="w-4 h-4 text-[#7c6cf0]" />;
    if (id === 'payment-svc') return <CreditCard className="w-4 h-4 text-[#b08cff]" />;
    if (id === 'inventory-svc') return <Database className="w-4 h-4 text-[#60a5fa]" />;
    if (id === 'notifications-svc') return <Bell className="w-4 h-4 text-[#818cf8]" />;
    if (id === 'user-service') return <Server className="w-4 h-4 text-[#4d8dff]" />;
    return <Box className="w-4 h-4 text-[#f3f4f8]" />;
  };

  return (
    <div className="flex flex-col gap-5 w-full animate-in fade-in duration-200">
      {/* Header Block */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-1">
        <div>
          <h1 className="text-[28px] font-semibold tracking-tight text-[#f3f4f8]">
            Service Map
          </h1>
          <p className="text-[13px] text-[#9aa0b5] mt-1">
            Visualize service dependencies and real-time health across your system.
          </p>
        </div>

        {/* Top Controls */}
        <div className="flex items-center gap-3">
          <Select
            options={[
              { value: '15m', label: 'Last 15 minutes' },
              { value: '1h', label: 'Last 1 hour' },
              { value: '24h', label: 'Last 24 hours' },
            ]}
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
            className="w-40"
          />

          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="w-9 h-9 rounded-[8px] bg-[#0a0a0d] border border-white/[0.08] hover:border-white/[0.14] text-[#9aa0b5] hover:text-[#f3f4f8] flex items-center justify-center transition-colors"
            title="Refresh graph"
          >
            <RefreshCw className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
          </button>

          <button
            type="button"
            className="w-9 h-9 rounded-[8px] bg-[#0a0a0d] border border-white/[0.08] hover:border-white/[0.14] text-[#9aa0b5] hover:text-[#f3f4f8] flex items-center justify-center transition-colors"
            title="Expand fullscreen"
          >
            <Maximize2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Main Two-Column View: Canvas (~70%) + Service Details (~30%) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
        {/* Left: Topology Canvas (Cols 8) */}
        <div className="lg:col-span-8 rounded-[14px] bg-[#050508] border border-white/[0.08] h-[520px] relative overflow-hidden flex flex-col justify-between shadow-inner">
          {/* SVG Canvas */}
          <svg className="w-full h-full select-none" viewBox="0 0 700 480">
            <defs>
              {/* Dotted Grid Pattern */}
              <pattern
                id="service-map-dots"
                width="24"
                height="24"
                patternUnits="userSpaceOnUse"
              >
                <circle cx="2" cy="2" r="1" fill="rgba(255, 255, 255, 0.08)" />
              </pattern>

              {/* Edge Gradient */}
              <linearGradient id="edge-flow-grad" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#7c6cf0" stopOpacity="0.8" />
                <stop offset="100%" stopColor="#3b82f6" stopOpacity="0.8" />
              </linearGradient>
            </defs>

            {/* Background Grid */}
            <rect width="100%" height="100%" fill="url(#service-map-dots)" />

            {/* Curved Bezier Edges */}
            {edges.map((edge) => {
              const srcNode = nodes.find((n) => n.id === edge.source);
              const tgtNode = nodes.find((n) => n.id === edge.target);
              if (!srcNode || !tgtNode) return null;

              const mx = (srcNode.x + tgtNode.x) / 2;
              const pathD = `M ${srcNode.x} ${srcNode.y} C ${mx} ${srcNode.y}, ${mx} ${tgtNode.y}, ${tgtNode.x} ${tgtNode.y}`;

              const isEdgeHighlighted =
                selectedNodeId === edge.source || selectedNodeId === edge.target;

              return (
                <g key={edge.id} className="transition-opacity duration-200">
                  {/* Background Path */}
                  <path
                    d={pathD}
                    fill="none"
                    stroke={
                      isEdgeHighlighted
                        ? '#7c6cf0'
                        : 'rgba(255, 255, 255, 0.15)'
                    }
                    strokeWidth={isEdgeHighlighted ? '2' : '1.5'}
                    strokeDasharray={edge.source === 'client' ? '4 4' : 'none'}
                  />

                  {/* Flowing Traffic Particle Dot */}
                  <circle r={isEdgeHighlighted ? '3.5' : '2.5'} fill="#b08cff">
                    <animateMotion
                      path={pathD}
                      dur={`${Math.max(1.8, 300 / edge.rps)}s`}
                      repeatCount="indefinite"
                    />
                  </circle>
                </g>
              );
            })}

            {/* Nodes */}
            {nodes.map((node) => {
              const isSelected = selectedNodeId === node.id;
              const isCritical = node.health === 'critical';
              const isDegraded = node.health === 'degraded';

              let ringColor = 'rgba(255, 255, 255, 0.15)';
              if (isSelected) ringColor = '#7c6cf0';
              else if (isCritical) ringColor = '#9b5cf6';
              else if (isDegraded) ringColor = '#818cf8';

              return (
                <g
                  key={node.id}
                  transform={`translate(${node.x}, ${node.y})`}
                  onClick={() => setSelectedNodeId(node.id)}
                  className="cursor-pointer group"
                >
                  {/* Subtle Glow Ring on selection or critical */}
                  {(isSelected || isCritical) && (
                    <circle
                      r="26"
                      fill="none"
                      stroke={ringColor}
                      strokeWidth="1.5"
                      strokeOpacity="0.4"
                      className="animate-pulse"
                    />
                  )}

                  {/* Node Base Circle */}
                  <circle
                    r="20"
                    fill="#0a0a0d"
                    stroke={ringColor}
                    strokeWidth={isSelected ? '2.5' : '1.5'}
                    className="transition-colors group-hover:stroke-[#7c6cf0]"
                  />

                  {/* Center Icon */}
                  <foreignObject x="-10" y="-10" width="20" height="20">
                    <div className="w-full h-full flex items-center justify-center">
                      {getNodeIcon(node.type, node.id)}
                    </div>
                  </foreignObject>

                  {/* Node Label Below */}
                  <text
                    y="36"
                    textAnchor="middle"
                    fill={isSelected ? '#ffffff' : '#f3f4f8'}
                    fontSize="12"
                    fontWeight={isSelected ? '600' : '500'}
                    fontFamily="Inter, sans-serif"
                    className="select-none"
                  >
                    {node.name}
                  </text>
                </g>
              );
            })}
          </svg>

          {/* Bottom Bar Info */}
          <div className="p-3 bg-black/90 border-t border-white/[0.08] flex items-center justify-between text-[11px] text-[#6b7188] px-4">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-[#60a5fa] animate-pulse" />
              <span>Real-time Topology • 7 Active Services</span>
            </div>
            <span>Click any node to inspect telemetry</span>
          </div>
        </div>

        {/* Right: Service Details Card (Cols 4) */}
        <div className="lg:col-span-4 rounded-[14px] bg-[#0a0a0d] border border-white/[0.08] p-5 flex flex-col gap-5">
          <div className="flex items-center justify-between pb-3 border-b border-white/[0.06]">
            <h2 className="text-[14px] font-semibold text-[#f3f4f8]">
              Service Details
            </h2>
            <span
              className={`px-2 py-0.5 rounded-full text-[10px] font-medium uppercase tracking-wider ${
                selectedNode.health === 'critical'
                  ? 'bg-[#9b5cf6]/20 text-[#b4a7f5] border border-[#9b5cf6]/40'
                  : selectedNode.health === 'degraded'
                  ? 'bg-[#818cf8]/20 text-[#818cf8] border border-[#818cf8]/40'
                  : 'bg-[#3b82f6]/20 text-[#60a5fa] border border-[#3b82f6]/40'
              }`}
            >
              {selectedNode.health}
            </span>
          </div>

          {/* Service Name Header Tile */}
          <div className="p-3.5 rounded-[10px] bg-white/[0.02] border border-[rgba(255,255,255,0.05)] flex items-center gap-3">
            <div className="w-10 h-10 rounded-[8px] bg-[#7c6cf0]/20 border border-[#7c6cf0]/40 flex items-center justify-center text-[#7c6cf0] shrink-0">
              <Box className="w-5 h-5" />
            </div>
            <div className="min-w-0">
              <div className="text-[15px] font-semibold text-[#f3f4f8] truncate">
                {selectedNode.name}
              </div>
              <div className="text-[11px] font-mono text-[#6b7188] capitalize">
                {selectedNode.type} tier
              </div>
            </div>
          </div>

          {/* 3 Metric Rows with Live Sparklines matching mockup */}
          <div className="flex flex-col gap-3">
            {/* 1. p99 Latency */}
            <div className="p-3.5 rounded-[10px] bg-white/[0.02] border border-[rgba(255,255,255,0.05)] flex items-center justify-between">
              <div>
                <span className="text-[11px] font-medium text-[#9aa0b5]">p99 Latency</span>
                <div className="text-[18px] font-bold text-[#f3f4f8] mt-0.5 font-mono">
                  {selectedNode.p99} ms
                </div>
              </div>
              <Sparkline
                data={selectedNode.sparklines.latency}
                width={110}
                height={32}
                color="#7c6cf0"
              />
            </div>

            {/* 2. Error Rate */}
            <div className="p-3.5 rounded-[10px] bg-white/[0.02] border border-[rgba(255,255,255,0.05)] flex items-center justify-between">
              <div>
                <span className="text-[11px] font-medium text-[#9aa0b5]">Error Rate</span>
                <div className="text-[18px] font-bold text-[#f3f4f8] mt-0.5 font-mono">
                  {selectedNode.errorRate}%
                </div>
              </div>
              <Sparkline
                data={selectedNode.sparklines.errorRate}
                width={110}
                height={32}
                color="#7c6cf0"
              />
            </div>

            {/* 3. Traces */}
            <div className="p-3.5 rounded-[10px] bg-white/[0.02] border border-[rgba(255,255,255,0.05)] flex items-center justify-between">
              <div>
                <span className="text-[11px] font-medium text-[#9aa0b5]">Traces</span>
                <div className="text-[18px] font-bold text-[#f3f4f8] mt-0.5 font-mono">
                  {selectedNode.traces.toLocaleString()}
                </div>
              </div>
              <Sparkline
                data={selectedNode.sparklines.traces}
                width={110}
                height={32}
                color="#7c6cf0"
              />
            </div>
          </div>

          {/* Quick Subtext */}
          <div className="pt-2 text-[11px] text-[#6b7188] border-t border-[rgba(255,255,255,0.05)] flex items-center justify-between">
            <span className="flex items-center gap-1 text-[#9aa0b5]">
              <Zap className="w-3 h-3 text-[#7c6cf0]" />
              Auto-sync: 10s
            </span>
            <span>ClickHouse v24.3</span>
          </div>
        </div>
      </div>
    </div>
  );
};
