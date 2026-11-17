import type { TraceRow, TraceDetail, Anomaly } from './api';

export const DEMO_KPIS = {
  totalTraces: '12,438',
  totalTracesDelta: 14.2,
  p99Latency: '67 ms',
  p99LatencyDelta: -12.4, // lower is better
  activeServices: 7,
  activeAnomalies: 3,
};

export const DEMO_LIVE_TRACES: TraceRow[] = [
  {
    trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
    root_service: 'checkout-api',
    root_operation: 'process_checkout',
    duration_ms: 124,
    span_count: 5,
    error_count: 0,
    start_time: Date.now() - 4000,
  },
  {
    trace_id: 'f8e7d6c5b4a392817263544536271809',
    root_service: 'payment-svc',
    root_operation: 'charge_stripe',
    duration_ms: 342,
    span_count: 3,
    error_count: 0,
    start_time: Date.now() - 12000,
  },
  {
    trace_id: '1928374655a4b3c2d1e0f9e8d7c6b5a4',
    root_service: 'inventory-svc',
    root_operation: 'check_stock',
    duration_ms: 45,
    span_count: 2,
    error_count: 0,
    start_time: Date.now() - 25000,
  },
  {
    trace_id: 'd4e5f60718a1b2c3d4e5f60718a1b2c3',
    root_service: 'notification-svc',
    root_operation: 'send_receipt',
    duration_ms: 89,
    span_count: 4,
    error_count: 1,
    start_time: Date.now() - 48000,
  },
];

export const DEMO_ANOMALIES_STATS = {
  total: 24,
  critical: 6,
  high: 8,
  medium: 10,
};

export interface AnomalyTableRow extends Anomaly {
  issue_description?: string;
  deviation_text?: string;
  time_ago?: string;
}

export const DEMO_ANOMALIES: AnomalyTableRow[] = [
  {
    id: 'anom-9041',
    org_id: 'org-default',
    service_name: 'payment-svc',
    operation_name: 'charge_card',
    root_cause_service: 'payment-svc',
    severity: 'critical',
    z_score: 4.8,
    baseline_latency_ms: 45.0,
    observed_latency_ms: 495.0,
    error_rate_delta: 0.12,
    status: 'open',
    detected_at: new Date(Date.now() - 2 * 60000).toISOString(),
    root_cause_path: ['frontend', 'checkout-api', 'payment-svc'],
    issue_description: 'DB connection pool exhaustion',
    deviation_text: '+450 ms',
    time_ago: '2m ago',
  },
  {
    id: 'anom-9042',
    org_id: 'org-default',
    service_name: 'inventory-svc',
    operation_name: 'reserve_items',
    root_cause_service: 'inventory-svc',
    severity: 'high',
    z_score: 3.2,
    baseline_latency_ms: 30.0,
    observed_latency_ms: 880.0,
    error_rate_delta: 0.18,
    status: 'open',
    detected_at: new Date(Date.now() - 8 * 60000).toISOString(),
    root_cause_path: ['checkout-api', 'inventory-svc'],
    issue_description: 'Redis cache eviction spike',
    deviation_text: '+850 ms',
    time_ago: '8m ago',
  },
  {
    id: 'anom-9043',
    org_id: 'org-default',
    service_name: 'api-gateway',
    operation_name: 'route_request',
    root_cause_service: 'api-gateway',
    severity: 'medium',
    z_score: 2.4,
    baseline_latency_ms: 10.0,
    observed_latency_ms: 130.0,
    error_rate_delta: 0.02,
    status: 'investigating',
    detected_at: new Date(Date.now() - 15 * 60000).toISOString(),
    root_cause_path: ['client', 'api-gateway'],
    issue_description: 'DNS resolution latency',
    deviation_text: '+120 ms',
    time_ago: '15m ago',
  },
  {
    id: 'anom-9044',
    org_id: 'org-default',
    service_name: 'checkout-api',
    operation_name: 'process_checkout',
    root_cause_service: 'checkout-api',
    severity: 'high',
    z_score: 3.8,
    baseline_latency_ms: 60.0,
    observed_latency_ms: 380.0,
    error_rate_delta: 0.052,
    status: 'open',
    detected_at: new Date(Date.now() - 22 * 60000).toISOString(),
    root_cause_path: ['api-gateway', 'checkout-api'],
    issue_description: 'Elevated error rate (5.2%)',
    deviation_text: '+320 ms',
    time_ago: '22m ago',
  },
];

export interface TopologyNode {
  id: string;
  name: string;
  type: 'client' | 'gateway' | 'service';
  x: number;
  y: number;
  p99: number;
  errorRate: number;
  traces: number;
  health: 'healthy' | 'degraded' | 'critical';
  sparklines: {
    latency: number[];
    errorRate: number[];
    traces: number[];
  };
}

export interface TopologyEdge {
  id: string;
  source: string;
  target: string;
  rps: number;
}

export const DEMO_TOPOLOGY_NODES: TopologyNode[] = [
  {
    id: 'client',
    name: 'Client',
    type: 'client',
    x: 80,
    y: 240,
    p99: 420,
    errorRate: 0.1,
    traces: 12438,
    health: 'healthy',
    sparklines: {
      latency: [350, 370, 390, 410, 420, 400, 420],
      errorRate: [0.1, 0.1, 0.2, 0.1, 0.1],
      traces: [11000, 11500, 12000, 12438],
    },
  },
  {
    id: 'api-gateway',
    name: 'api-gateway',
    type: 'gateway',
    x: 230,
    y: 240,
    p99: 210,
    errorRate: 0.2,
    traces: 8940,
    health: 'healthy',
    sparklines: {
      latency: [180, 195, 200, 210, 205, 210],
      errorRate: [0.1, 0.2, 0.2, 0.2],
      traces: [8000, 8400, 8700, 8940],
    },
  },
  {
    id: 'user-service',
    name: 'user-service',
    type: 'service',
    x: 390,
    y: 130,
    p99: 120,
    errorRate: 0.1,
    traces: 3420,
    health: 'healthy',
    sparklines: {
      latency: [110, 115, 120, 118, 120],
      errorRate: [0.1, 0.1, 0.1],
      traces: [3000, 3200, 3420],
    },
  },
  {
    id: 'checkout-api',
    name: 'checkout-api',
    type: 'service',
    x: 390,
    y: 240,
    p99: 380,
    errorRate: 0.8,
    traces: 2431,
    health: 'degraded',
    sparklines: {
      latency: [240, 260, 290, 310, 340, 360, 350, 380],
      errorRate: [0.2, 0.3, 0.4, 0.5, 0.4, 0.7, 0.8],
      traces: [1800, 1950, 2100, 2250, 2300, 2431],
    },
  },
  {
    id: 'notifications-svc',
    name: 'notifications-svc',
    type: 'service',
    x: 390,
    y: 350,
    p99: 65,
    errorRate: 0.05,
    traces: 1850,
    health: 'healthy',
    sparklines: {
      latency: [55, 60, 62, 65, 63, 65],
      errorRate: [0.05, 0.05],
      traces: [1600, 1750, 1850],
    },
  },
  {
    id: 'payment-svc',
    name: 'payment-svc',
    type: 'service',
    x: 580,
    y: 160,
    p99: 450,
    errorRate: 3.2,
    traces: 1620,
    health: 'critical',
    sparklines: {
      latency: [180, 220, 280, 350, 410, 450],
      errorRate: [0.5, 1.2, 2.0, 3.2],
      traces: [1200, 1400, 1550, 1620],
    },
  },
  {
    id: 'inventory-svc',
    name: 'inventory-svc',
    type: 'service',
    x: 580,
    y: 320,
    p99: 85,
    errorRate: 0.4,
    traces: 2190,
    health: 'healthy',
    sparklines: {
      latency: [60, 70, 75, 80, 85],
      errorRate: [0.2, 0.3, 0.4],
      traces: [1900, 2050, 2190],
    },
  },
];

export const DEMO_TOPOLOGY_EDGES: TopologyEdge[] = [
  { id: 'e1', source: 'client', target: 'api-gateway', rps: 142 },
  { id: 'e2', source: 'api-gateway', target: 'user-service', rps: 38 },
  { id: 'e3', source: 'api-gateway', target: 'checkout-api', rps: 84 },
  { id: 'e4', source: 'api-gateway', target: 'notifications-svc', rps: 20 },
  { id: 'e5', source: 'checkout-api', target: 'payment-svc', rps: 42 },
  { id: 'e6', source: 'checkout-api', target: 'inventory-svc', rps: 65 },
];

export const DEMO_WATERFALL_SPANS = [
  {
    id: 'span-1',
    service: 'frontend',
    operation: 'POST /checkout',
    startMs: 0,
    durationMs: 120,
    depth: 0,
  },
  {
    id: 'span-2',
    service: 'payment-svc',
    operation: 'POST /charge',
    startMs: 25,
    durationMs: 85,
    depth: 1,
    isHot: true,
  },
  {
    id: 'span-3',
    service: 'inventory-svc',
    operation: 'POST /reserve',
    startMs: 35,
    durationMs: 45,
    depth: 2,
  },
  {
    id: 'span-4',
    service: 'notification-svc',
    operation: 'POST /send',
    startMs: 90,
    durationMs: 20,
    depth: 1,
  },
  {
    id: 'span-5',
    service: 'database-svc',
    operation: 'COMMIT transaction',
    startMs: 110,
    durationMs: 10,
    depth: 2,
  },
];

export const DEMO_SERVICE_HEALTH_SEGMENTS = [
  { label: 'Healthy', value: 5, color: '#3b82f6' },
  { label: 'Degraded', value: 1, color: '#818cf8' },
  { label: 'Unhealthy', value: 1, color: '#7c6cf0' },
];

export const DEMO_TRACES_LIST: TraceRow[] = [
  {
    trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
    root_service: 'checkout-api',
    root_operation: 'process_checkout',
    duration_ms: 124,
    span_count: 5,
    error_count: 0,
    start_time: Date.now() - 35000,
  },
  {
    trace_id: 'f8e7d6c5b4a392817263544536271809',
    root_service: 'payment-svc',
    root_operation: 'charge_stripe',
    duration_ms: 342,
    span_count: 3,
    error_count: 0,
    start_time: Date.now() - 110000,
  },
  {
    trace_id: '1928374655a4b3c2d1e0f9e8d7c6b5a4',
    root_service: 'inventory-svc',
    root_operation: 'check_stock',
    duration_ms: 45,
    span_count: 2,
    error_count: 0,
    start_time: Date.now() - 190000,
  },
  {
    trace_id: 'd4e5f60718a1b2c3d4e5f60718a1b2c3',
    root_service: 'notification-svc',
    root_operation: 'send_receipt',
    duration_ms: 89,
    span_count: 4,
    error_count: 1,
    start_time: Date.now() - 280000,
  },
  {
    trace_id: '7b8a9c0d1e2f34567890abcdef123456',
    root_service: 'order-router',
    root_operation: 'dispatch_order',
    duration_ms: 215,
    span_count: 6,
    error_count: 0,
    start_time: Date.now() - 420000,
  },
  {
    trace_id: '99887766554433221100aabbccddeeff',
    root_service: 'auth-svc',
    root_operation: 'refresh_session',
    duration_ms: 18,
    span_count: 2,
    error_count: 0,
    start_time: Date.now() - 550000,
  },
  {
    trace_id: 'eeff00112233445566778899aabbccdd',
    root_service: 'shipping-api',
    root_operation: 'calculate_rates',
    duration_ms: 540,
    span_count: 4,
    error_count: 1,
    start_time: Date.now() - 670000,
  },
];

export const DEMO_TRACE_DETAILS: Record<string, TraceDetail> = {
  a1b2c3d4e5f60718293a4b5c6d7e8f90: {
    trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
    root_service: 'checkout-api',
    root_operation: 'process_checkout',
    duration_ms: 124,
    span_count: 5,
    error_count: 0,
    start_time: Date.now() - 35000,
    spans: [
      {
        trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
        span_id: '1111222233334444',
        parent_span_id: '',
        service_name: 'frontend',
        operation_name: 'POST /checkout',
        duration_ms: 124,
        status_code: 1,
        start_time: new Date(Date.now() - 35000).toISOString(),
        start_offset_ms: 0,
        tags: {
          'http.method': 'POST',
          'http.target': '/checkout',
          'http.status_code': '200',
          'http.user_agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
          'user.id': 'usr_994821',
          'client.ip': '192.0.2.45',
        },
      },
      {
        trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
        span_id: '2222333344445555',
        parent_span_id: '1111222233334444',
        service_name: 'payment-svc',
        operation_name: 'POST /charge',
        duration_ms: 85,
        status_code: 1,
        start_time: new Date(Date.now() - 34975).toISOString(),
        start_offset_ms: 25,
        tags: {
          'payment.gateway': 'stripe',
          'payment.amount': '149.00',
          'payment.currency': 'USD',
          'db.system': 'postgresql',
          'db.statement': 'INSERT INTO ledger_entries (id, amount) VALUES ($1, $2)',
        },
      },
      {
        trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
        span_id: '3333444455556666',
        parent_span_id: '2222333344445555',
        service_name: 'inventory-svc',
        operation_name: 'POST /reserve',
        duration_ms: 45,
        status_code: 1,
        start_time: new Date(Date.now() - 34965).toISOString(),
        start_offset_ms: 35,
        tags: {
          'inventory.item_count': '3',
          'inventory.warehouse': 'us-east-1a',
          'db.system': 'redis',
        },
      },
      {
        trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
        span_id: '4444555566667777',
        parent_span_id: '1111222233334444',
        service_name: 'notification-svc',
        operation_name: 'POST /send',
        duration_ms: 20,
        status_code: 1,
        start_time: new Date(Date.now() - 34910).toISOString(),
        start_offset_ms: 90,
        tags: {
          'notification.channel': 'email',
          'notification.template': 'order_receipt_v2',
        },
      },
      {
        trace_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
        span_id: '5555666677778888',
        parent_span_id: '2222333344445555',
        service_name: 'database-svc',
        operation_name: 'COMMIT transaction',
        duration_ms: 10,
        status_code: 1,
        start_time: new Date(Date.now() - 34890).toISOString(),
        start_offset_ms: 110,
        tags: {
          'db.transaction_id': 'tx_99214',
          'db.rows_affected': '1',
        },
      },
    ],
  },
};

export const DEMO_COSTS_KPIS = {
  totalMonthlyCost: '$1,248.60',
  avgCostPerIncident: '$0.42',
  llmTokenSpend: '$386.20',
  infrastructure: '$862.40',
};

export interface CostTrendPoint {
  date: string;
  cost: number;
}

export const DEMO_COST_TREND: CostTrendPoint[] = [
  { date: 'Nov 10', cost: 280 },
  { date: 'Nov 11', cost: 340 },
  { date: 'Nov 12', cost: 410 },
  { date: 'Nov 13', cost: 360 },
  { date: 'Nov 14', cost: 490 },
  { date: 'Nov 15', cost: 520 },
  { date: 'Nov 16', cost: 480 },
  { date: 'Nov 17', cost: 560 },
  { date: 'Nov 18', cost: 620 },
  { date: 'Nov 19', cost: 710 },
  { date: 'Nov 20', cost: 680 },
  { date: 'Nov 21', cost: 780 },
  { date: 'Nov 22', cost: 840 },
];

export interface CostByServiceItem {
  id: string;
  service: string;
  percentage: number;
  cost: string;
}

export const DEMO_COST_BY_SERVICE: CostByServiceItem[] = [
  { id: 'payment-svc', service: 'payment-svc', percentage: 38, cost: '$474.47' },
  { id: 'inventory-svc', service: 'inventory-svc', percentage: 24, cost: '$299.66' },
  { id: 'checkout-api', service: 'checkout-api', percentage: 16, cost: '$199.78' },
  { id: 'api-gateway', service: 'api-gateway', percentage: 12, cost: '$149.83' },
  { id: 'notifications-svc', service: 'notifications-svc', percentage: 10, cost: '$124.86' },
];

