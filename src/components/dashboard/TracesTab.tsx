import React, { useState, useMemo } from 'react';
import { Search, ChevronDown, ChevronRight, AlertCircle, Clock, Filter, X } from 'lucide-react';

// --- Types ---
interface Span {
  id: string;
  parentId?: string;
  service: string;
  operation: string;
  duration: number; // ms
  startOffset: number; // ms from trace start
  error?: boolean;
  tags?: Record<string, string>;
}

interface Trace {
  id: string;
  rootService: string;
  rootOperation: string;
  duration: number;
  spanCount: number;
  errorCount: number;
  startTime: number; // timestamp
  spans: Span[];
}

// --- Mock Data Generator ---
const SERVICES = ['checkout-api', 'payment-svc', 'inventory-svc', 'user-service', 'api-gateway', 'notifications-svc'];
const OPERATIONS: Record<string, string[]> = {
  'api-gateway': ['GET /api/checkout', 'POST /api/payment', 'GET /api/users'],
  'checkout-api': ['process_checkout', 'validate_cart', 'calculate_tax'],
  'payment-svc': ['charge_card', 'verify_funds', 'refund'],
  'inventory-svc': ['check_stock', 'reserve_items', 'update_inventory'],
  'user-service': ['get_user', 'validate_session'],
  'notifications-svc': ['send_email', 'send_sms'],
};

const generateId = () => Math.random().toString(16).substring(2, 18);

const generateSpans = (traceId: string, duration: number, errorCount: number): Span[] => {
  const rootService = SERVICES[Math.floor(Math.random() * SERVICES.length)];
  const rootOp = OPERATIONS[rootService][Math.floor(Math.random() * OPERATIONS[rootService].length)];
  const rootSpanId = generateId();
  
  const spans: Span[] = [
    {
      id: rootSpanId,
      service: rootService,
      operation: rootOp,
      duration: duration,
      startOffset: 0,
      error: errorCount > 0,
    }
  ];

  const spanCount = Math.floor(Math.random() * 8) + 3; // 3 to 10 spans
  let currentParentId = rootSpanId;
  let currentOffset = 5;

  for (let i = 1; i < spanCount; i++) {
    const srv = SERVICES[Math.floor(Math.random() * SERVICES.length)];
    const op = OPERATIONS[srv][Math.floor(Math.random() * OPERATIONS[srv].length)];
    const spanDuration = Math.max(10, Math.floor(Math.random() * (duration - currentOffset - 10)));
    const isError = i < errorCount + 1; // distribute errors

    spans.push({
      id: generateId(),
      parentId: currentParentId,
      service: srv,
      operation: op,
      duration: spanDuration,
      startOffset: currentOffset,
      error: isError,
    });

    currentOffset += Math.floor(Math.random() * 20) + 5;
    if (Math.random() > 0.5) {
      currentParentId = spans[spans.length - 1].id;
    }
  }

  return spans;
};

const mockTraces: Trace[] = Array.from({ length: 150 }).map(() => {
  const duration = Math.floor(Math.random() * 1500) + 50;
  const isError = Math.random() > 0.8;
  const errorCount = isError ? Math.floor(Math.random() * 3) + 1 : 0;
  const spans = generateSpans(generateId(), duration, errorCount);
  const rootSpan = spans[0];
  
  return {
    id: generateId(),
    rootService: rootSpan.service,
    rootOperation: rootSpan.operation,
    duration,
    spanCount: spans.length,
    errorCount,
    startTime: Date.now() - Math.floor(Math.random() * 3600000), // last 1 hour
    spans,
  };
});

// --- Components ---

const FilterSelect: React.FC<{ label: string, options: string[], value: string, onChange: (v: string) => void }> = ({ label, options, value, onChange }) => (
  <div className="flex flex-col gap-1">
    <label className="text-[10px] uppercase text-white/50 tracking-wider">{label}</label>
    <div className="relative">
      <select 
        value={value} 
        onChange={e => onChange(e.target.value)}
        className="appearance-none bg-white/[0.04] border border-white/10 rounded px-3 py-1.5 text-sm text-white pr-8 focus:outline-none focus:border-white/30 cursor-pointer w-full"
      >
        <option value="all">All</option>
        {options.map(o => <option key={o} value={o}>{o}</option>)}
      </select>
      <ChevronDown className="absolute right-2 top-2 w-4 h-4 text-white/40 pointer-events-none" />
    </div>
  </div>
);

const WaterfallNode: React.FC<{ span: Span, traceDuration: number, depth: number, allSpans: Span[] }> = ({ span, traceDuration, depth, allSpans }) => {
  const children = allSpans.filter(s => s.parentId === span.id);
  const [expanded, setExpanded] = useState(true);
  
  const leftPct = (span.startOffset / traceDuration) * 100;
  const widthPct = Math.max(0.5, (span.duration / traceDuration) * 100);

  return (
    <div className="flex flex-col font-dm-mono text-sm">
      <div className="flex items-center hover:bg-white/[0.02] py-1 border-b border-white/[0.02] group relative">
        <div className="w-[300px] shrink-0 flex items-center gap-2" style={{ paddingLeft: `${depth * 16 + 8}px` }}>
          {children.length > 0 ? (
            <button onClick={() => setExpanded(!expanded)} className="text-white/40 hover:text-white">
              <ChevronRight className={`w-3 h-3 transition-transform ${expanded ? 'rotate-90' : ''}`} />
            </button>
          ) : <div className="w-3" />}
          
          <div className="flex flex-col overflow-hidden">
            <span className={`truncate ${span.error ? 'text-red-400' : 'text-white'}`}>{span.service}</span>
            <span className="text-xs text-white/50 truncate">{span.operation}</span>
          </div>
        </div>
        
        <div className="flex-1 relative h-6 flex items-center pr-4">
          <div className="absolute inset-0 border-l border-white/[0.05] pointer-events-none" />
          <div 
            className={`h-3 rounded-sm relative group-hover:opacity-100 transition-opacity ${span.error ? 'bg-red-500/80' : 'bg-blue-500/80'}`}
            style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
          >
            <div className="absolute top-full mt-1 left-0 bg-black border border-white/10 text-xs px-2 py-1 rounded opacity-0 group-hover:opacity-100 whitespace-nowrap z-10 pointer-events-none">
              {span.duration}ms
            </div>
          </div>
        </div>
      </div>
      
      {expanded && children.map(child => (
        <WaterfallNode key={child.id} span={child} traceDuration={traceDuration} depth={depth + 1} allSpans={allSpans} />
      ))}
    </div>
  );
};

export const TracesTab: React.FC = () => {
  const [search, setSearch] = useState('');
  const [service, setService] = useState('all');
  const [status, setStatus] = useState('all');
  const [expandedTraceId, setExpandedTraceId] = useState<string | null>(null);
  
  const [sortField, setSortField] = useState<keyof Trace>('startTime');
  const [sortAsc, setSortAsc] = useState(false);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const filtered = useMemo(() => {
    let res = mockTraces;
    if (search) res = res.filter(t => t.id.includes(search) || t.rootService.includes(search));
    if (service !== 'all') res = res.filter(t => t.rootService === service);
    if (status !== 'all') res = res.filter(t => status === 'error' ? t.errorCount > 0 : t.errorCount === 0);
    
    res.sort((a, b) => {
      const valA = a[sortField];
      const valB = b[sortField];
      if (valA < valB) return sortAsc ? -1 : 1;
      if (valA > valB) return sortAsc ? 1 : -1;
      return 0;
    });
    
    return res;
  }, [search, service, status, sortField, sortAsc]);

  const maxDuration = useMemo(() => Math.max(...filtered.map(t => t.duration)), [filtered]);
  const paginated = filtered.slice((page - 1) * pageSize, page * pageSize);

  const handleSort = (field: keyof Trace) => {
    if (sortField === field) setSortAsc(!sortAsc);
    else {
      setSortField(field);
      setSortAsc(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-black">
      {/* FILTER BAR */}
      <div className="p-4 border-b border-white/[0.08] flex flex-wrap gap-4 items-end bg-white/[0.02]">
        <div className="flex-1 min-w-[200px]">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 w-4 h-4 text-white/40" />
            <input 
              type="text" 
              placeholder="Search trace ID or service..." 
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full bg-white/[0.04] border border-white/10 rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder:text-white/30 focus:outline-none focus:border-white/30 font-dm-mono transition-colors"
            />
          </div>
        </div>
        <FilterSelect label="Service" options={SERVICES} value={service} onChange={setService} />
        <FilterSelect label="Status" options={['success', 'error']} value={status} onChange={setStatus} />
        
        <div className="flex items-center gap-2 text-sm text-white/50 bg-white/[0.04] border border-white/10 rounded px-3 py-1.5 h-[34px]">
          <Clock className="w-4 h-4" />
          <span>Last 1 hour</span>
        </div>
      </div>

      {/* TABLE */}
      <div className="flex-1 overflow-auto relative">
        <table className="w-full text-left border-collapse min-w-[800px]">
          <thead className="sticky top-0 bg-black border-b border-white/[0.08] z-20">
            <tr>
              {['Trace ID', 'Root Service', 'Duration', 'Spans', 'Errors', 'Start Time'].map((col, i) => (
                <th key={col} className="px-4 py-3 text-[11px] uppercase tracking-wider text-white/40 font-medium whitespace-nowrap cursor-pointer hover:text-white" onClick={() => handleSort(col.toLowerCase().includes('time') ? 'startTime' : col.toLowerCase().includes('id') ? 'id' : col.toLowerCase().includes('service') ? 'rootService' : col.toLowerCase().includes('span') ? 'spanCount' : col.toLowerCase().includes('error') ? 'errorCount' : 'duration')}>
                  {col} {sortField.toLowerCase().includes(col.split(' ')[0].toLowerCase()) && (sortAsc ? '↑' : '↓')}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-white/[0.04]">
            {paginated.map(trace => (
              <React.Fragment key={trace.id}>
                <tr 
                  onClick={() => setExpandedTraceId(expandedTraceId === trace.id ? null : trace.id)}
                  className={`hover:bg-white/[0.02] cursor-pointer transition-colors ${expandedTraceId === trace.id ? 'bg-white/[0.02]' : ''}`}
                >
                  <td className="px-4 py-3">
                    <span className="font-dm-mono text-sm text-blue-400 hover:underline">{trace.id}</span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-col">
                      <span className={`text-sm ${trace.errorCount > 0 ? 'text-red-400' : 'text-white'}`}>{trace.rootService}</span>
                      <span className="text-xs text-white/40 truncate w-48">{trace.rootOperation}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 w-48">
                    <div className="flex items-center gap-2">
                      <span className="font-dm-mono text-xs w-12">{trace.duration}ms</span>
                      <div className="flex-1 h-1.5 bg-white/[0.05] rounded-full overflow-hidden">
                        <div className={`h-full rounded-full ${trace.errorCount > 0 ? 'bg-red-500' : 'bg-blue-500'}`} style={{ width: `${(trace.duration / maxDuration) * 100}%` }} />
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-white/70">{trace.spanCount}</td>
                  <td className="px-4 py-3">
                    {trace.errorCount > 0 ? (
                      <span className="inline-flex items-center gap-1 text-xs text-red-400 bg-red-400/10 px-2 py-0.5 rounded">
                        <AlertCircle className="w-3 h-3" /> {trace.errorCount}
                      </span>
                    ) : <span className="text-sm text-white/30">0</span>}
                  </td>
                  <td className="px-4 py-3 font-dm-mono text-xs text-white/50">
                    {Math.floor((Date.now() - trace.startTime) / 60000)}m ago
                  </td>
                </tr>
                {expandedTraceId === trace.id && (
                  <tr>
                    <td colSpan={6} className="p-0 border-b-2 border-white/10">
                      <div className="bg-black border-y border-white/[0.05] p-4 max-h-[400px] overflow-y-auto">
                        <div className="text-xs text-white/40 uppercase tracking-widest mb-4">Trace Waterfall</div>
                        <WaterfallNode span={trace.spans[0]} traceDuration={trace.duration} depth={0} allSpans={trace.spans} />
                      </div>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            ))}
          </tbody>
        </table>
        
        {paginated.length === 0 && (
          <div className="flex flex-col items-center justify-center h-48 text-white/40">
            <Filter className="w-8 h-8 mb-2 opacity-50" />
            <p>No traces match the current filters</p>
          </div>
        )}
      </div>

      {/* PAGINATION */}
      <div className="p-4 border-t border-white/[0.08] flex items-center justify-between text-sm text-white/50 bg-black">
        <span>Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of {filtered.length} traces</span>
        <div className="flex items-center gap-2">
          <button disabled={page === 1} onClick={() => setPage(p => p - 1)} className="px-3 py-1 hover:bg-white/[0.05] rounded disabled:opacity-30">Prev</button>
          <span className="px-3 py-1 bg-white/[0.05] rounded">Page {page}</span>
          <button disabled={page * pageSize >= filtered.length} onClick={() => setPage(p => p + 1)} className="px-3 py-1 hover:bg-white/[0.05] rounded disabled:opacity-30">Next</button>
        </div>
      </div>
    </div>
  );
};
