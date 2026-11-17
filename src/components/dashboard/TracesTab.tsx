import React, { useState, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  RefreshCw,
  Send,
  Loader2,
  Copy,
  Check,
  Layers,
} from 'lucide-react';
import { tracesApi, servicesApi, spansApi } from '../../lib/api';
import type { TraceRow, TraceDetail } from '../../lib/api';
import { DataTable, type Column } from '../common/DataTable';
import { Select } from '../common/Select';
import { Tabs } from '../common/Tabs';
import { StatusPill } from '../common/StatusPill';
import { Pagination } from '../common/Pagination';
import { EmptyState } from '../common/EmptyState';
import { Skeleton } from '../common/Skeleton';
import { DEMO_TRACES_LIST, DEMO_TRACE_DETAILS } from '../../lib/fixtures';

export const TracesTab: React.FC = () => {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  // Search & Filters from URL or state
  const selectedTraceId = searchParams.get('trace') || '';
  const [searchText, setSearchText] = useState(searchParams.get('search') || '');
  const [serviceFilter, setServiceFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [timeRange, setTimeRange] = useState<'15m' | '1h' | '24h'>('1h');
  const [page, setPage] = useState(1);
  const pageSize = 10;

  // Inspector state
  const [activeInspectorTab, setActiveInspectorTab] = useState<'spans' | 'tags' | 'json'>('spans');
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);
  const [copiedTraceId, setCopiedTraceId] = useState(false);
  const [copiedJson, setCopiedJson] = useState(false);

  // Queries
  const { data: serviceOptions = [] } = useQuery({
    queryKey: ['services-options'],
    queryFn: () => servicesApi.list(),
  });

  const {
    data: tracesData = [],
    isLoading: isLoadingTraces,
    isFetching: isFetchingTraces,
    refetch: refetchTraces,
  } = useQuery({
    queryKey: ['traces-list', serviceFilter, statusFilter],
    queryFn: () => tracesApi.search({ service: serviceFilter, status: statusFilter, limit: 100 }),
  });

  const tracesList: TraceRow[] =
    tracesData && tracesData.length > 0 ? tracesData : DEMO_TRACES_LIST;

  // Filtered traces
  const filteredTraces = useMemo(() => {
    let result = [...tracesList];
    if (searchText.trim()) {
      const q = searchText.toLowerCase().trim();
      result = result.filter(
        (t) =>
          t.trace_id.toLowerCase().includes(q) ||
          t.root_service.toLowerCase().includes(q) ||
          t.root_operation.toLowerCase().includes(q)
      );
    }
    return result;
  }, [tracesList, searchText]);

  // Selected trace detail query
  const { data: traceDetailData, isLoading: isLoadingDetail } = useQuery({
    queryKey: ['trace-detail', selectedTraceId],
    queryFn: () => tracesApi.get(selectedTraceId),
    enabled: !!selectedTraceId,
  });

  // Effective trace detail (with fixture fallback)
  const activeTraceDetail: TraceDetail | null = useMemo(() => {
    if (!selectedTraceId) return null;
    if (traceDetailData) return traceDetailData;
    if (DEMO_TRACE_DETAILS[selectedTraceId]) return DEMO_TRACE_DETAILS[selectedTraceId];
    // Synthesize detail from list if available
    const found = tracesList.find((t) => t.trace_id === selectedTraceId);
    if (found) {
      return {
        ...found,
        spans: [
          {
            trace_id: found.trace_id,
            span_id: 'span-root-001',
            parent_span_id: '',
            service_name: found.root_service,
            operation_name: found.root_operation,
            duration_ms: found.duration_ms,
            status_code: found.error_count > 0 ? 2 : 1,
            start_time: new Date(found.start_time).toISOString(),
            start_offset_ms: 0,
            tags: {
              'service.name': found.root_service,
              'operation.name': found.root_operation,
              'span.kind': 'SERVER',
            },
          },
        ],
      };
    }
    return null;
  }, [selectedTraceId, traceDetailData, tracesList]);

  // Active span for inspector
  const activeSpan = useMemo(() => {
    if (!activeTraceDetail || !activeTraceDetail.spans?.length) return null;
    if (selectedSpanId) {
      const found = activeTraceDetail.spans.find((s) => s.span_id === selectedSpanId);
      if (found) return found;
    }
    return activeTraceDetail.spans[0];
  }, [activeTraceDetail, selectedSpanId]);

  // Ingest sample telemetry mutation
  const ingestMutation = useMutation({
    mutationFn: async () => {
      const traceId = Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      const rootSpanId = Array.from({ length: 16 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      const childSpanId = Array.from({ length: 16 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      const now = new Date().toISOString();

      return spansApi.ingest([
        {
          trace_id: traceId,
          span_id: rootSpanId,
          service_name: 'api-gateway',
          operation_name: 'GET /api/checkout',
          duration_ms: 120,
          status_code: 1,
          start_time: now,
          tags: { 'http.method': 'GET', 'http.status_code': '200' },
        },
        {
          trace_id: traceId,
          span_id: childSpanId,
          parent_span_id: rootSpanId,
          service_name: 'payment-svc',
          operation_name: 'charge_credit_card',
          duration_ms: 85,
          status_code: 1,
          start_time: now,
          start_offset_ms: 25,
          tags: { 'payment.provider': 'stripe' },
        },
      ]);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['traces-list'] });
    },
  });

  const handleSelectTrace = (traceId: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('trace', traceId);
      return next;
    });
    setSelectedSpanId(null);
  };

  const handleCopyTraceId = () => {
    if (!selectedTraceId) return;
    navigator.clipboard.writeText(selectedTraceId);
    setCopiedTraceId(true);
    setTimeout(() => setCopiedTraceId(false), 2000);
  };

  const handleCopyJson = () => {
    if (!activeTraceDetail) return;
    navigator.clipboard.writeText(JSON.stringify(activeTraceDetail, null, 2));
    setCopiedJson(true);
    setTimeout(() => setCopiedJson(false), 2000);
  };

  // Pagination slice
  const totalPages = Math.ceil(filteredTraces.length / pageSize) || 1;
  const paginatedTraces = filteredTraces.slice((page - 1) * pageSize, page * pageSize);

  // Columns definition
  const columns: Column<TraceRow>[] = [
    {
      key: 'trace_id',
      header: 'Trace ID',
      width: '180px',
      render: (t) => (
        <span className="font-mono text-[13px] text-[#4d8dff] hover:underline">
          {t.trace_id.slice(0, 8)}...{t.trace_id.slice(-6)}
        </span>
      ),
    },
    {
      key: 'root_service',
      header: 'Service',
      render: (t) => (
        <div className="flex flex-col">
          <span className="text-[13px] font-medium text-[#f3f4f8]">{t.root_service}</span>
          <span className="font-mono text-[11px] text-[#6b7188] truncate max-w-[150px]">
            {t.root_operation}
          </span>
        </div>
      ),
    },
    {
      key: 'duration_ms',
      header: 'Duration',
      render: (t) => (
        <div className="flex items-center gap-2">
          <span className="font-mono text-[12px] text-[#f3f4f8] w-12">{t.duration_ms}ms</span>
          <div className="w-16 h-1.5 rounded-full bg-white/[0.06] overflow-hidden">
            <div
              className={`h-full rounded-full ${
                t.error_count > 0
                  ? 'bg-red-500'
                  : 'bg-gradient-to-r from-[#3b6cf0] to-[#5b8cff]'
              }`}
              style={{
                width: `${Math.min(100, Math.max(10, (t.duration_ms / 400) * 100))}%`,
              }}
            />
          </div>
        </div>
      ),
    },
    {
      key: 'span_count',
      header: 'Spans',
      align: 'center',
      render: (t) => <span className="font-mono text-[12px] text-[#9aa0b5]">{t.span_count}</span>,
    },
    {
      key: 'error_count',
      header: 'Status',
      align: 'center',
      render: (t) => (
        <StatusPill status={t.error_count > 0 ? 'error' : 'ok'} size="sm" />
      ),
    },
    {
      key: 'start_time',
      header: 'Time',
      align: 'right',
      render: (t) => (
        <span className="text-[11px] font-mono text-[#6b7188]">
          {t.start_time
            ? `${Math.max(0, Math.floor((Date.now() - t.start_time) / 60000))}m ago`
            : 'just now'}
        </span>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-5 w-full">
      {/* Top Filter Bar */}
      <div className="p-4 rounded-[14px] bg-[#0a0a0d] border border-white/[0.08] flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3 flex-1 min-w-[280px]">
          {/* Search box */}
          <div className="relative min-w-[240px] flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#6b7188]" />
            <input
              type="text"
              placeholder="Filter by trace ID, service..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              className="w-full h-9 pl-9 pr-4 rounded-[8px] bg-[#050508] text-[#f3f4f8] text-[13px] border border-white/[0.08] placeholder:text-[#6b7188] focus:outline-none focus:border-[#7c6cf0] focus:ring-1 focus:ring-[#7c6cf0] transition-colors"
            />
          </div>

          {/* Service Dropdown */}
          <Select
            options={[
              { value: 'all', label: 'All Services' },
              ...serviceOptions.map((s) => ({ value: s, label: s })),
            ]}
            value={serviceFilter}
            onChange={(e) => setServiceFilter(e.target.value)}
            className="w-40"
          />

          {/* Status Dropdown */}
          <Select
            options={[
              { value: 'all', label: 'All Statuses' },
              { value: 'ok', label: 'Ok (Success)' },
              { value: 'error', label: 'Error' },
            ]}
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="w-36"
          />

          {/* Time range pills */}
          <div className="flex items-center rounded-[8px] bg-[#050508] p-1 border border-white/[0.08]">
            {(['15m', '1h', '24h'] as const).map((r) => (
              <button
                key={r}
                type="button"
                onClick={() => setTimeRange(r)}
                className={`px-2.5 py-1 text-[11px] font-medium rounded-[6px] transition-colors ${
                  timeRange === r
                    ? 'bg-[#7c6cf0]/20 text-[#f3f4f8] font-semibold'
                    : 'text-[#9aa0b5] hover:text-[#f3f4f8]'
                }`}
              >
                {r}
              </button>
            ))}
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => refetchTraces()}
            disabled={isFetchingTraces}
            className="h-9 px-3 rounded-[8px] bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.08] text-[12px] font-medium text-[#f3f4f8] flex items-center gap-2 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isFetchingTraces ? 'animate-spin' : ''}`} />
            Refresh
          </button>

          <button
            onClick={() => ingestMutation.mutate()}
            disabled={ingestMutation.isPending}
            className="h-9 px-3 rounded-[8px] bg-[#7c6cf0]/15 hover:bg-[#7c6cf0]/25 border border-[#7c6cf0]/30 text-[12px] font-medium text-[#b4a7f5] flex items-center gap-2 transition-colors disabled:opacity-50"
            title="Ingest test sample spans"
          >
            {ingestMutation.isPending ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Send className="w-3.5 h-3.5" />
            )}
            Send Span
          </button>
        </div>
      </div>

      {/* Two-Pane Main Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
        {/* Left Pane: Traces DataTable (~55% width, cols 7) */}
        <div className="lg:col-span-7 flex flex-col gap-3">
          <div className="flex items-center justify-between px-1">
            <span className="text-[13px] font-medium text-[#9aa0b5]">
              Showing <span className="text-[#f3f4f8] font-semibold">{filteredTraces.length}</span> traces
            </span>
          </div>

          <DataTable
            columns={columns}
            data={paginatedTraces}
            keyExtractor={(t) => t.trace_id}
            selectedKey={selectedTraceId}
            onRowClick={(t) => handleSelectTrace(t.trace_id)}
            loading={isLoadingTraces}
            emptyMessage="No traces matching current filters."
          />

          {totalPages > 1 && (
            <div className="flex justify-center mt-2">
              <Pagination
                currentPage={page}
                totalPages={totalPages}
                onPageChange={(p) => setPage(p)}
              />
            </div>
          )}
        </div>

        {/* Right Pane: Trace Detail (~45% width, cols 5) */}
        <div className="lg:col-span-5 rounded-[14px] bg-[#0a0a0d] border border-white/[0.08] p-5 min-h-[500px] flex flex-col">
          {!selectedTraceId ? (
            <div className="m-auto py-12">
              <EmptyState
                icon={<Layers className="w-8 h-8 text-[#6b7188]" />}
                title="No Trace Selected"
                description="Select a trace from the left panel to inspect its waterfall, span hierarchy, and attributes."
              />
            </div>
          ) : isLoadingDetail && !activeTraceDetail ? (
            <div className="flex flex-col gap-4">
              <Skeleton height="32px" width="60%" />
              <Skeleton height="80px" />
              <Skeleton height="150px" />
            </div>
          ) : activeTraceDetail ? (
            <div className="flex flex-col gap-4">
              {/* Header card with trace metadata */}
              <div className="p-4 rounded-[10px] bg-white/[0.02] border border-white/[0.06] flex flex-col gap-2.5">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-[11px] font-medium uppercase tracking-wider text-[#6b7188]">
                      Trace Detail
                    </span>
                    <StatusPill
                      status={activeTraceDetail.error_count > 0 ? 'error' : 'ok'}
                      size="sm"
                    />
                  </div>

                  <button
                    onClick={handleCopyTraceId}
                    className="flex items-center gap-1.5 px-2 py-1 rounded-[6px] text-[11px] font-mono text-[#9aa0b5] hover:text-[#f3f4f8] bg-white/[0.04] hover:bg-white/[0.08] transition-colors"
                    title="Copy full trace ID"
                  >
                    {copiedTraceId ? (
                      <>
                        <Check className="w-3 h-3 text-[#60a5fa]" />
                        <span>Copied</span>
                      </>
                    ) : (
                      <>
                        <Copy className="w-3 h-3" />
                        <span>Copy ID</span>
                      </>
                    )}
                  </button>
                </div>

                <div className="font-mono text-[13px] text-[#4d8dff] break-all select-all">
                  {activeTraceDetail.trace_id}
                </div>

                <div className="flex flex-wrap items-center gap-3 pt-2 border-t border-[rgba(255,255,255,0.05)] text-[12px] text-[#9aa0b5]">
                  <span className="font-medium text-[#f3f4f8]">
                    {activeTraceDetail.root_service}
                  </span>
                  <span>•</span>
                  <span className="font-mono">{activeTraceDetail.duration_ms} ms</span>
                  <span>•</span>
                  <span>{activeTraceDetail.spans.length} spans</span>
                  {activeTraceDetail.error_count > 0 && (
                    <>
                      <span>•</span>
                      <span className="text-red-400 font-medium">
                        {activeTraceDetail.error_count} error(s)
                      </span>
                    </>
                  )}
                </div>
              </div>

              {/* Waterfall Timeline */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <span className="text-[12px] font-semibold text-[#f3f4f8]">
                    Span Timeline ({activeTraceDetail.spans.length})
                  </span>
                  <span className="text-[11px] text-[#6b7188]">
                    Click span to view tags & logs
                  </span>
                </div>

                <div className="flex flex-col divide-y divide-white/[0.04] border border-white/[0.08] rounded-[8px] bg-[#050508] overflow-hidden">
                  {activeTraceDetail.spans.map((s) => {
                    const maxDur = activeTraceDetail.duration_ms || 1;
                    const offset = s.start_offset_ms || 0;
                    const leftPct = Math.min(100, Math.max(0, (offset / maxDur) * 100));
                    const widthPct = Math.min(
                      100 - leftPct,
                      Math.max(2, (s.duration_ms / maxDur) * 100)
                    );
                    const isSelected = activeSpan?.span_id === s.span_id;
                    const hasError = s.status_code === 2 || !!s.error_message;

                    return (
                      <div
                        key={s.span_id}
                        onClick={() => setSelectedSpanId(s.span_id)}
                        className={`p-2.5 flex items-center justify-between cursor-pointer transition-colors ${
                          isSelected
                            ? 'bg-[#7c6cf0]/15 border-l-2 border-[#7c6cf0]'
                            : 'hover:bg-white/[0.02]'
                        }`}
                      >
                        <div className="flex flex-col min-w-0 pr-3">
                          <span
                            className={`text-[12px] font-medium truncate ${
                              hasError ? 'text-red-400' : 'text-[#f3f4f8]'
                            }`}
                          >
                            {s.service_name}
                          </span>
                          <span className="font-mono text-[11px] text-[#6b7188] truncate">
                            {s.operation_name}
                          </span>
                        </div>

                        {/* Bar */}
                        <div className="flex items-center gap-2 w-36 shrink-0">
                          <div className="flex-1 h-2 rounded-full bg-white/[0.05] relative overflow-hidden">
                            <div
                              className={`absolute top-0 bottom-0 rounded-full ${
                                hasError
                                  ? 'bg-red-500'
                                  : isSelected
                                  ? 'bg-[#9b6bf2]'
                                  : 'bg-[#4d8dff]'
                              }`}
                              style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
                            />
                          </div>
                          <span className="font-mono text-[11px] text-[#9aa0b5] w-10 text-right">
                            {s.duration_ms}ms
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Inspector Tabs */}
              <div className="mt-2 flex flex-col gap-3">
                <Tabs
                  variant="underline"
                  items={[
                    { id: 'spans', label: 'Spans', count: activeTraceDetail.spans.length },
                    { id: 'tags', label: 'Tags & Attributes' },
                    { id: 'json', label: 'Raw JSON' },
                  ]}
                  activeId={activeInspectorTab}
                  onChange={(id) => setActiveInspectorTab(id as 'spans' | 'tags' | 'json')}
                />

                {/* Tab: Spans List */}
                {activeInspectorTab === 'spans' && (
                  <div className="flex flex-col gap-2 max-h-60 overflow-y-auto">
                    {activeTraceDetail.spans.map((sp) => (
                      <div
                        key={sp.span_id}
                        onClick={() => setSelectedSpanId(sp.span_id)}
                        className={`p-2.5 rounded-[8px] border text-[12px] flex items-center justify-between cursor-pointer transition-colors ${
                          activeSpan?.span_id === sp.span_id
                            ? 'bg-white/[0.04] border-[#7c6cf0]/40'
                            : 'bg-white/[0.01] border-[rgba(255,255,255,0.04)] hover:border-white/10'
                        }`}
                      >
                        <div className="flex flex-col">
                          <span className="font-medium text-[#f3f4f8]">{sp.service_name}</span>
                          <span className="font-mono text-[11px] text-[#6b7188]">{sp.operation_name}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-[11px] text-[#9aa0b5]">{sp.duration_ms}ms</span>
                          <StatusPill status={sp.status_code === 2 ? 'error' : 'ok'} size="sm" />
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* Tab: Tags & Attributes */}
                {activeInspectorTab === 'tags' && (
                  <div className="flex flex-col gap-2 max-h-60 overflow-y-auto">
                    {activeSpan?.tags && Object.keys(activeSpan.tags).length > 0 ? (
                      <div className="rounded-[8px] border border-white/[0.08] bg-[#050508] overflow-hidden text-[12px]">
                        <table className="w-full text-left">
                          <tbody className="divide-y divide-white/[0.04] font-mono text-[11px]">
                            {Object.entries(activeSpan.tags).map(([k, v]) => (
                              <tr key={k} className="hover:bg-white/[0.02]">
                                <td className="p-2.5 text-[#9aa0b5] font-semibold w-1/3 select-all">
                                  {k}
                                </td>
                                <td className="p-2.5 text-[#f3f4f8] break-all select-all">{v}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : (
                      <div className="text-[12px] text-[#6b7188] py-4 text-center">
                        No tags or attributes reported for this span.
                      </div>
                    )}
                  </div>
                )}

                {/* Tab: Raw JSON */}
                {activeInspectorTab === 'json' && (
                  <div className="relative">
                    <button
                      onClick={handleCopyJson}
                      className="absolute right-3 top-3 z-10 px-2 py-1 rounded text-[11px] font-mono bg-white/10 hover:bg-white/20 text-[#f3f4f8] flex items-center gap-1 transition-colors"
                    >
                      {copiedJson ? <Check className="w-3 h-3 text-[#60a5fa]" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedJson ? 'Copied' : 'Copy'}</span>
                    </button>
                    <pre className="p-3 rounded-[8px] bg-[#050508] border border-white/[0.08] text-[11px] font-mono text-[#9aa0b5] overflow-auto max-h-60 select-all">
                      {JSON.stringify(activeTraceDetail, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
};
