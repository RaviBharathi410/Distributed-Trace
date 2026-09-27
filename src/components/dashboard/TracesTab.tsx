import React, { useState, useMemo } from 'react';
import { Search, ChevronDown, ChevronRight, AlertCircle, Clock, Filter, RefreshCw, Send, Loader2 } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { tracesApi, servicesApi, spansApi } from '../../lib/api';
import type { TraceRow, Span } from '../../lib/api';

// --- Components ---

const FilterSelect: React.FC<{ label: string; options: string[]; value: string; onChange: (v: string) => void }> = ({
  label,
  options,
  value,
  onChange,
}) => (
  <div className="flex flex-col gap-1">
    <label className="text-[10px] uppercase text-white/50 tracking-wider">{label}</label>
    <div className="relative">
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="appearance-none bg-white/[0.04] border border-white/10 rounded px-3 py-1.5 text-sm text-white pr-8 focus:outline-none focus:border-white/30 cursor-pointer w-full"
      >
        <option value="all">All</option>
        {options.map((o) => (
          <option key={o} value={o}>
            {o}
          </option>
        ))}
      </select>
      <ChevronDown className="absolute right-2 top-2 w-4 h-4 text-white/40 pointer-events-none" />
    </div>
  </div>
);

const WaterfallNode: React.FC<{
  span: Span;
  traceDuration: number;
  depth: number;
  allSpans: Span[];
}> = ({ span, traceDuration, depth, allSpans }) => {
  const children = allSpans.filter((s) => s.parent_span_id === span.span_id);
  const [expanded, setExpanded] = useState(true);

  const startOffset = span.start_offset_ms || 0;
  const leftPct = traceDuration > 0 ? (startOffset / traceDuration) * 100 : 0;
  const widthPct = traceDuration > 0 ? Math.max(1, (span.duration_ms / traceDuration) * 100) : 100;
  const isError = span.status_code === 2 || !!span.error_message;

  return (
    <div className="flex flex-col font-dm-mono text-sm">
      <div className="flex items-center hover:bg-white/[0.02] py-1 border-b border-white/[0.02] group relative">
        <div className="w-[320px] shrink-0 flex items-center gap-2" style={{ paddingLeft: `${depth * 16 + 8}px` }}>
          {children.length > 0 ? (
            <button
              onClick={() => setExpanded(!expanded)}
              className="text-white/40 hover:text-white"
              aria-label="Toggle child spans"
            >
              <ChevronRight className={`w-3 h-3 transition-transform ${expanded ? 'rotate-90' : ''}`} />
            </button>
          ) : (
            <div className="w-3" />
          )}

          <div className="flex flex-col overflow-hidden">
            <span className={`truncate ${isError ? 'text-red-400 font-semibold' : 'text-white'}`}>{span.service_name}</span>
            <span className="text-xs text-white/50 truncate">{span.operation_name}</span>
          </div>
        </div>

        <div className="flex-1 relative h-6 flex items-center pr-4">
          <div className="absolute inset-0 border-l border-white/[0.05] pointer-events-none" />
          <div
            className={`h-3 rounded-sm relative group-hover:opacity-100 transition-opacity ${
              isError ? 'bg-red-500/80' : 'bg-blue-500/80'
            }`}
            style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
          >
            <div className="absolute top-full mt-1 left-0 bg-black border border-white/10 text-xs px-2 py-1 rounded opacity-0 group-hover:opacity-100 whitespace-nowrap z-10 pointer-events-none shadow-lg">
              {span.duration_ms}ms {isError && span.error_message ? `• ${span.error_message}` : ''}
            </div>
          </div>
        </div>
      </div>

      {expanded &&
        children.map((child) => (
          <WaterfallNode
            key={child.span_id}
            span={child}
            traceDuration={traceDuration}
            depth={depth + 1}
            allSpans={allSpans}
          />
        ))}
    </div>
  );
};

export const TracesTab: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [service, setService] = useState('all');
  const [status, setStatus] = useState('all');
  const [expandedTraceId, setExpandedTraceId] = useState<string | null>(null);

  const [sortField, setSortField] = useState<keyof TraceRow>('start_time');
  const [sortAsc, setSortAsc] = useState(false);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // Real React Query: Services
  const { data: serviceOptions = [] } = useQuery({
    queryKey: ['services'],
    queryFn: () => servicesApi.list(),
  });

  // Real React Query: Traces list
  const {
    data: traces = [],
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery({
    queryKey: ['traces', service, status],
    queryFn: () => tracesApi.search({ service, status, limit: 100 }),
  });

  // Real React Query: Expanded trace detail
  const { data: activeTraceDetail, isLoading: isLoadingDetail } = useQuery({
    queryKey: ['trace-detail', expandedTraceId],
    queryFn: () => tracesApi.get(expandedTraceId!),
    enabled: !!expandedTraceId,
  });

  const [pipelineFlushing, setPipelineFlushing] = useState(false);

  // Ingest sample telemetry mutation
  const ingestMutation = useMutation({
    mutationFn: async () => {
      const traceId = Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      const rootSpanId = Array.from({ length: 16 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      const childSpanId = Array.from({ length: 16 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      const now = new Date().toISOString();

      const sampleBatch = [
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
      ];
      return spansApi.ingest(sampleBatch);
    },
    onSuccess: () => {
      setPipelineFlushing(true);
      queryClient.invalidateQueries({ queryKey: ['traces'] });
      queryClient.invalidateQueries({ queryKey: ['services'] });

      // Staggered invalidations to smoothly catch Kafka batch consumer flush window (~1s)
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['traces'] });
        queryClient.invalidateQueries({ queryKey: ['services'] });
        setPipelineFlushing(false);
      }, 1200);

      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['traces'] });
        queryClient.invalidateQueries({ queryKey: ['services'] });
      }, 2500);
    },
  });

  const filtered = useMemo(() => {
    let res = [...traces];
    if (search) {
      res = res.filter((t) => t.trace_id.includes(search) || t.root_service.toLowerCase().includes(search.toLowerCase()));
    }

    res.sort((a, b) => {
      const valA = a[sortField];
      const valB = b[sortField];
      if (valA < valB) return sortAsc ? -1 : 1;
      if (valA > valB) return sortAsc ? 1 : -1;
      return 0;
    });

    return res;
  }, [traces, search, sortField, sortAsc]);

  const maxDuration = useMemo(
    () => (filtered.length > 0 ? Math.max(...filtered.map((t) => t.duration_ms || 1)) : 100),
    [filtered],
  );
  const paginated = filtered.slice((page - 1) * pageSize, page * pageSize);

  const handleSort = (field: keyof TraceRow) => {
    if (sortField === field) {
      setSortAsc(!sortAsc);
    } else {
      setSortField(field);
      setSortAsc(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-black text-white">
      {/* FILTER BAR */}
      <div className="p-4 border-b border-white/[0.08] flex flex-wrap gap-4 items-end bg-white/[0.02]">
        <div className="flex-1 min-w-[200px]">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 w-4 h-4 text-white/40" />
            <input
              type="text"
              placeholder="Search trace ID or service..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-white/[0.04] border border-white/10 rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder:text-white/30 focus:outline-none focus:border-white/30 font-dm-mono transition-colors"
            />
          </div>
        </div>
        <FilterSelect
          label="Service"
          options={serviceOptions.length > 0 ? serviceOptions : ['api-gateway', 'checkout-api', 'payment-svc']}
          value={service}
          onChange={setService}
        />
        <FilterSelect label="Status" options={['success', 'error']} value={status} onChange={setStatus} />

        <div className="flex items-center gap-2 text-sm text-white/50 bg-white/[0.04] border border-white/10 rounded px-3 py-1.5 h-[34px]">
          <Clock className="w-4 h-4" />
          <span>Last 1 hour</span>
        </div>

        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="flex items-center gap-2 text-xs text-white/70 hover:text-white bg-white/[0.06] hover:bg-white/[0.1] border border-white/10 rounded px-3 py-2 h-[34px] transition-colors"
          title="Refresh traces"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          <span>Refresh</span>
        </button>

        <button
          onClick={() => ingestMutation.mutate()}
          disabled={ingestMutation.isPending || pipelineFlushing}
          className="flex items-center gap-2 text-xs font-medium text-emerald-300 bg-emerald-950/40 hover:bg-emerald-900/60 border border-emerald-500/30 rounded px-3 py-2 h-[34px] transition-colors disabled:opacity-60"
          title="Send real test telemetry batch to POST /api/v1/spans"
        >
          {ingestMutation.isPending || pipelineFlushing ? (
            <>
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
              <span>{ingestMutation.isPending ? 'Sending...' : 'Queued (Flushing...)'}</span>
            </>
          ) : (
            <>
              <Send className="w-3.5 h-3.5" />
              <span>Send Sample Span</span>
            </>
          )}
        </button>
      </div>

      {/* ERROR BANNER IF BACKEND OFFLINE */}
      {isError && (
        <div className="m-4 p-4 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-amber-400 shrink-0" />
            <div className="text-sm">
              <span className="font-semibold text-amber-300">Live Backend Offline or Database Unreachable:</span>{' '}
              <span className="text-white/70">{(error as Error)?.message || 'Could not connect to http://localhost:8080'}</span>
            </div>
          </div>
          <button
            onClick={() => refetch()}
            className="text-xs bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 px-3 py-1.5 rounded border border-amber-500/40 transition-colors"
          >
            Retry Connection
          </button>
        </div>
      )}

      {/* TABLE */}
      <div className="flex-1 overflow-auto relative">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center h-64 text-white/50 gap-3">
            <Loader2 className="w-6 h-6 animate-spin text-blue-400" />
            <span className="text-sm">Loading telemetry traces from ClickHouse...</span>
          </div>
        ) : (
          <table className="w-full text-left border-collapse min-w-[800px]">
            <thead className="sticky top-0 bg-black border-b border-white/[0.08] z-20">
              <tr>
                {[
                  { label: 'Trace ID', key: 'trace_id' as keyof TraceRow },
                  { label: 'Root Service', key: 'root_service' as keyof TraceRow },
                  { label: 'Duration', key: 'duration_ms' as keyof TraceRow },
                  { label: 'Spans', key: 'span_count' as keyof TraceRow },
                  { label: 'Errors', key: 'error_count' as keyof TraceRow },
                  { label: 'Start Time', key: 'start_time' as keyof TraceRow },
                ].map((col) => (
                  <th
                    key={col.label}
                    className="px-4 py-3 text-[11px] uppercase tracking-wider text-white/40 font-medium whitespace-nowrap cursor-pointer hover:text-white"
                    onClick={() => handleSort(col.key)}
                  >
                    {col.label} {sortField === col.key && (sortAsc ? '↑' : '↓')}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-white/[0.04]">
              {paginated.map((trace) => (
                <React.Fragment key={trace.trace_id}>
                  <tr
                    onClick={() => setExpandedTraceId(expandedTraceId === trace.trace_id ? null : trace.trace_id)}
                    className={`hover:bg-white/[0.02] cursor-pointer transition-colors ${
                      expandedTraceId === trace.trace_id ? 'bg-white/[0.02]' : ''
                    }`}
                  >
                    <td className="px-4 py-3">
                      <span className="font-dm-mono text-sm text-blue-400 hover:underline">{trace.trace_id}</span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-col">
                        <span className={`text-sm ${trace.error_count > 0 ? 'text-red-400' : 'text-white'}`}>
                          {trace.root_service}
                        </span>
                        <span className="text-xs text-white/40 truncate w-48">{trace.root_operation}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 w-48">
                      <div className="flex items-center gap-2">
                        <span className="font-dm-mono text-xs w-12">{trace.duration_ms}ms</span>
                        <div className="flex-1 h-1.5 bg-white/[0.05] rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full ${trace.error_count > 0 ? 'bg-red-500' : 'bg-blue-500'}`}
                            style={{ width: `${Math.min(100, (trace.duration_ms / maxDuration) * 100)}%` }}
                          />
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-white/70">{trace.span_count}</td>
                    <td className="px-4 py-3">
                      {trace.error_count > 0 ? (
                        <span className="inline-flex items-center gap-1 text-xs text-red-400 bg-red-400/10 px-2 py-0.5 rounded">
                          <AlertCircle className="w-3 h-3" /> {trace.error_count}
                        </span>
                      ) : (
                        <span className="text-sm text-white/30">0</span>
                      )}
                    </td>
                    <td className="px-4 py-3 font-dm-mono text-xs text-white/50">
                      {trace.start_time ? `${Math.max(0, Math.floor((Date.now() - trace.start_time) / 60000))}m ago` : 'just now'}
                    </td>
                  </tr>
                  {expandedTraceId === trace.trace_id && (
                    <tr>
                      <td colSpan={6} className="p-0 border-b-2 border-white/10">
                        <div className="bg-black border-y border-white/[0.05] p-4 max-h-[400px] overflow-y-auto">
                          <div className="text-xs text-white/40 uppercase tracking-widest mb-4">Trace Waterfall</div>
                          {isLoadingDetail ? (
                            <div className="flex items-center gap-2 text-sm text-white/50 py-4">
                              <Loader2 className="w-4 h-4 animate-spin text-blue-400" />
                              <span>Loading spans from ClickHouse...</span>
                            </div>
                          ) : activeTraceDetail && activeTraceDetail.spans && activeTraceDetail.spans.length > 0 ? (
                            <WaterfallNode
                              span={activeTraceDetail.spans[0]}
                              traceDuration={activeTraceDetail.duration_ms || trace.duration_ms}
                              depth={0}
                              allSpans={activeTraceDetail.spans}
                            />
                          ) : (
                            <div className="text-xs text-white/40 py-2">No individual spans returned for this trace.</div>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              ))}
            </tbody>
          </table>
        )}

        {!isLoading && paginated.length === 0 && (
          <div className="flex flex-col items-center justify-center h-48 text-white/40 gap-2">
            <Filter className="w-8 h-8 opacity-50" />
            <p className="text-sm">No traces found for the current query.</p>
            <button
              onClick={() => ingestMutation.mutate()}
              disabled={ingestMutation.isPending || pipelineFlushing}
              className="mt-2 text-xs text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 px-3 py-1.5 rounded border border-emerald-500/30 transition-colors disabled:opacity-60 inline-flex items-center gap-1.5"
            >
              {ingestMutation.isPending || pipelineFlushing ? (
                <>
                  <Loader2 className="w-3 h-3 animate-spin" />
                  <span>{ingestMutation.isPending ? 'Sending Telemetry...' : 'Queued (Flushing to ClickHouse...)'}</span>
                </>
              ) : (
                'Send Sample Telemetry Span to Database'
              )}
            </button>
          </div>
        )}
      </div>

      {/* PAGINATION */}
      <div className="p-4 border-t border-white/[0.08] flex items-center justify-between text-sm text-white/50 bg-black">
        <span>
          Showing {filtered.length === 0 ? 0 : (page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of{' '}
          {filtered.length} traces
        </span>
        <div className="flex items-center gap-2">
          <button
            disabled={page === 1}
            onClick={() => setPage((p) => p - 1)}
            className="px-3 py-1 hover:bg-white/[0.05] rounded disabled:opacity-30"
          >
            Prev
          </button>
          <span className="px-3 py-1 bg-white/[0.05] rounded">Page {page}</span>
          <button
            disabled={page * pageSize >= filtered.length}
            onClick={() => setPage((p) => p + 1)}
            className="px-3 py-1 hover:bg-white/[0.05] rounded disabled:opacity-30"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
};
