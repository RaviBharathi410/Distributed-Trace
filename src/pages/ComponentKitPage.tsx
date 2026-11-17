import React, { useState } from 'react';
import {
  Card,
  StatCard,
  SeverityBadge,
  StatusPill,
  DataTable,
  Select,
  Tabs,
  Pagination,
  Sparkline,
  DonutChart,
  Waterfall,
  BarChart,
  HorizontalBars,
  Skeleton,
  EmptyState,
  ErrorBanner,
} from '../components/common';
import { Activity, ShieldAlert, Zap, Layers, RefreshCw } from 'lucide-react';

export const ComponentKitPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState('pill-1');
  const [selectedPage, setSelectedPage] = useState(2);
  const [selectVal, setSelectVal] = useState('1h');

  // Sample data for DataTable
  const sampleTableData = [
    {
      id: 'a1b2c3d4e5f60718',
      service: 'order-service',
      operation: 'POST /checkout',
      duration: 124,
      status: 'ok',
    },
    {
      id: '9f8e7d6c5b4a3210',
      service: 'payment-svc',
      operation: 'POST /charge',
      duration: 480,
      status: 'error',
    },
    {
      id: '1234567890abcdef',
      service: 'inventory-svc',
      operation: 'GET /items',
      duration: 42,
      status: 'ok',
    },
  ];

  // Sample data for Waterfall
  const sampleWaterfallSpans = [
    {
      id: 'span-1',
      service: 'frontend',
      operation: 'POST /api/checkout',
      startMs: 0,
      durationMs: 240,
      depth: 0,
    },
    {
      id: 'span-2',
      service: 'order-service',
      operation: 'create_order',
      startMs: 20,
      durationMs: 200,
      depth: 1,
    },
    {
      id: 'span-3',
      service: 'payment-svc',
      operation: 'process_payment',
      startMs: 60,
      durationMs: 140,
      depth: 2,
      isHot: true,
    },
    {
      id: 'span-4',
      service: 'notification-svc',
      operation: 'send_receipt',
      startMs: 210,
      durationMs: 25,
      depth: 1,
    },
  ];

  // Sample data for BarChart
  const sampleBarData = [
    { label: 'Oct 01', value: 1240 },
    { label: 'Oct 02', value: 1890 },
    { label: 'Oct 03', value: 2390 },
    { label: 'Oct 04', value: 3490, isHot: true },
    { label: 'Oct 05', value: 2100 },
    { label: 'Oct 06', value: 1600 },
    { label: 'Oct 07', value: 2780 },
  ];

  // Sample data for HorizontalBars
  const sampleHorizontalData = [
    { id: '1', label: 'payment-svc', value: 142.5, formattedValue: '$142.50', isHot: true },
    { id: '2', label: 'order-service', value: 89.2, formattedValue: '$89.20' },
    { id: '3', label: 'inventory-svc', value: 45.1, formattedValue: '$45.10' },
    { id: '4', label: 'auth-service', value: 21.0, formattedValue: '$21.00' },
  ];

  return (
    <div className="flex flex-col gap-10 w-full max-w-[1400px] mx-auto pb-16">
      {/* Page Title */}
      <div>
        <h1 className="text-[28px] font-semibold text-[#f3f4f8]">Design System Component Kit</h1>
        <p className="text-[13px] text-[#9aa0b5] mt-1">
          Complete gallery of shared design-system components, design tokens, and states for DistributedTrace.
        </p>
      </div>

      {/* 1. Design Tokens */}
      <Card title="1. Color Tokens & Eyedropped Palette" subtitle="Eyedropped from DistributedTrace_Dark-Mode_Dashboard_Suite.png">
        <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-4">
          {[
            { name: '--bg', color: '#000000', desc: 'Page background' },
            { name: '--card', color: '#0a0a0d', desc: 'Card / Panel fill' },
            { name: '--card-hover', color: '#111116', desc: 'Panel hover' },
            { name: '--accent', color: '#7c6cf0', desc: 'Violet brand / active' },
            { name: '--link', color: '#4d8dff', desc: 'Trace link mono' },
            { name: '--good', color: '#60a5fa', desc: 'Healthy / Positive' },
            { name: '--sev-critical', color: '#9b5cf6', desc: 'Critical badge' },
            { name: '--sev-high', color: '#3b82f6', desc: 'High badge' },
            { name: '--sev-medium', color: '#818cf8', desc: 'Medium badge' },
            { name: '--donut-healthy', color: '#3b82f6', desc: 'Healthy blue' },
            { name: '--bar-hot', color: '#9b6bf2', desc: 'Waterfall hot / highlight' },
            { name: '--text-2', color: '#9aa0b5', desc: 'Secondary label' },
          ].map((token) => (
            <div key={token.name} className="flex flex-col gap-1.5 p-3 rounded-[8px] bg-white/[0.02] border border-white/[0.05]">
              <div
                className="w-full h-10 rounded-[6px] border border-white/10 shadow-inner"
                style={{ backgroundColor: token.color }}
              />
              <span className="text-[11px] font-mono text-[#f3f4f8] font-medium">{token.name}</span>
              <span className="text-[10px] font-mono text-[#6b7188]">{token.color}</span>
              <span className="text-[10px] text-[#9aa0b5]">{token.desc}</span>
            </div>
          ))}
        </div>
      </Card>

      {/* 2. StatCards & KPI Tiles */}
      <div>
        <h2 className="text-[18px] font-semibold text-[#f3f4f8] mb-4">2. StatCards & KPI Cards</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="Total Traces"
            value="12,438"
            delta={14.2}
            deltaLabel="vs previous 24h"
            icon={<Activity className="w-5 h-5" />}
          />
          <StatCard
            label="p99 Latency"
            value="67 ms"
            delta={-12.4}
            invertDelta={true}
            deltaLabel="vs previous 24h"
            icon={<Zap className="w-5 h-5" />}
            iconColor="#60a5fa"
            iconBg="rgba(96, 165, 250, 0.12)"
          />
          <StatCard
            label="Active Services"
            value="7"
            subtext="All reporting healthy"
            icon={<Layers className="w-5 h-5" />}
            iconColor="#3b82f6"
            iconBg="rgba(59, 130, 246, 0.12)"
          />
          <StatCard
            label="Active Anomalies"
            value="3"
            subtext="1 critical, 2 medium"
            icon={<ShieldAlert className="w-5 h-5" />}
            iconColor="#818cf8"
            iconBg="rgba(129, 140, 248, 0.12)"
          />
        </div>
      </div>

      {/* 3. Badges & Status Pills */}
      <Card title="3. SeverityBadges & StatusPills" subtitle="Pre-configured accessible status indicators">
        <div className="flex flex-col gap-6">
          <div className="flex items-center gap-3 flex-wrap">
            <span className="text-[12px] text-[#9aa0b5] w-28">Severity:</span>
            <SeverityBadge severity="critical" />
            <SeverityBadge severity="high" />
            <SeverityBadge severity="medium" />
            <SeverityBadge severity="low" />
            <SeverityBadge severity="info" />
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            <span className="text-[12px] text-[#9aa0b5] w-28">Health:</span>
            <StatusPill status="healthy" />
            <StatusPill status="degraded" />
            <StatusPill status="critical" />
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            <span className="text-[12px] text-[#9aa0b5] w-28">Anomaly Status:</span>
            <StatusPill status="open" />
            <StatusPill status="investigating" />
            <StatusPill status="resolved" />
          </div>
        </div>
      </Card>

      {/* 4. Controls: Select, Tabs, Pagination */}
      <Card title="4. Form & Navigation Controls" subtitle="Select dropdown, pill & line tabs, pagination">
        <div className="flex flex-col gap-6">
          <div className="flex flex-wrap items-center gap-6">
            <Select
              label="Time Range"
              value={selectVal}
              onChange={(e) => setSelectVal(e.target.value)}
              options={[
                { value: '15m', label: 'Last 15 minutes' },
                { value: '1h', label: 'Last 1 hour' },
                { value: '24h', label: 'Last 24 hours' },
                { value: '7d', label: 'Last 7 days' },
              ]}
            />

            <div className="flex flex-col gap-1">
              <span className="text-[11px] font-medium text-[#9aa0b5]">Pill Tabs</span>
              <Tabs
                items={[
                  { id: 'pill-1', label: 'Overview' },
                  { id: 'pill-2', label: 'Latency' },
                  { id: 'pill-3', label: 'Errors', count: 4 },
                ]}
                activeId={activeTab}
                onChange={setActiveTab}
              />
            </div>
          </div>

          <div className="pt-2">
            <Pagination
              page={selectedPage}
              pageSize={10}
              totalItems={84}
              onPageChange={setSelectedPage}
            />
          </div>
        </div>
      </Card>

      {/* 5. DataTable */}
      <Card
        title="5. Generic Accessible DataTable"
        subtitle="Monospaced clickable IDs, status indicators, and clean cell padding"
        action={
          <button className="text-[12px] text-[#9aa0b5] hover:text-white flex items-center gap-1.5 px-2.5 py-1 rounded bg-white/5 border border-white/10">
            <RefreshCw className="w-3.5 h-3.5" />
            <span>Refresh</span>
          </button>
        }
      >
        <DataTable
          columns={[
            {
              key: 'id',
              header: 'Trace ID',
              render: (row) => (
                <span className="font-mono text-[#4d8dff] hover:underline cursor-pointer">
                  {row.id}
                </span>
              ),
            },
            { key: 'service', header: 'Root Service' },
            { key: 'operation', header: 'Operation' },
            {
              key: 'duration',
              header: 'Duration',
              align: 'right',
              render: (row) => <span className="font-mono">{row.duration} ms</span>,
            },
            {
              key: 'status',
              header: 'Status',
              render: (row) => <StatusPill status={row.status} />,
            },
          ]}
          data={sampleTableData}
          keyExtractor={(row) => row.id}
        />
      </Card>

      {/* 6. Visualizations: Waterfall, Donut, BarChart, HorizontalBars, Sparkline */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Waterfall */}
        <Card
          title="6A. Waterfall Timeline"
          subtitle="Hierarchical span duration representation with hot-path highlighting"
          className="lg:col-span-2"
        >
          <Waterfall spans={sampleWaterfallSpans} totalDurationMs={240} />
        </Card>

        {/* Donut Chart */}
        <Card title="6B. Service Health Donut" subtitle="Radial breakdown with center count">
          <DonutChart
            segments={[
              { label: 'Healthy', value: 5, color: '#3b82f6' },
              { label: 'Degraded', value: 1, color: '#818cf8' },
              { label: 'Unhealthy', value: 1, color: '#7c6cf0' },
            ]}
          />
        </Card>

        {/* Bar Chart */}
        <Card title="6C. Cost / Volume Bar Chart" subtitle="Vertical gradient bars with hover tooltips" className="lg:col-span-2">
          <BarChart data={sampleBarData} valuePrefix="$" />
        </Card>

        {/* Horizontal Bars & Sparklines */}
        <Card title="6D. Horizontal Breakdown & Sparklines" subtitle="Service spend / duration distribution">
          <HorizontalBars items={sampleHorizontalData} />

          <div className="mt-6 pt-4 border-t border-[rgba(255,255,255,0.06)] flex items-center justify-between">
            <span className="text-[12px] text-[#9aa0b5]">Mini Sparkline:</span>
            <Sparkline data={[12, 19, 14, 25, 22, 30, 28, 42, 36]} width={140} height={28} />
          </div>
        </Card>
      </div>

      {/* 7. Feedback & State Feedback: Skeleton, EmptyState, ErrorBanner */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card title="7A. ErrorBanner" subtitle="With message and retry callback">
          <ErrorBanner
            title="Failed to fetch live traces"
            message="ClickHouse server connection timeout (504). Please verify database reachability."
            onRetry={() => alert('Retrying fetch...')}
          />
        </Card>

        <Card title="7B. EmptyState" subtitle="Clean fallback when zero items exist">
          <EmptyState
            title="No anomalies detected"
            description="All services are operating within normal baseline bounds for the selected time window."
          />
        </Card>

        <Card title="7C. Skeletons" subtitle="Loading pulse placeholders" className="md:col-span-2">
          <div className="flex flex-col gap-3">
            <Skeleton height={20} width="40%" />
            <Skeleton height={40} width="100%" />
            <div className="grid grid-cols-3 gap-4 mt-2">
              <Skeleton height={60} rounded="lg" />
              <Skeleton height={60} rounded="lg" />
              <Skeleton height={60} rounded="lg" />
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
};
