// API client layer for DistributedTrace backend (REST API v1)

export interface User {
  id: string;
  email: string;
  org_id: string;
  role: 'owner' | 'admin' | 'member' | 'viewer';
  created_at: string;
  updated_at: string;
}

export interface Organization {
  id: string;
  name: string;
  plan: string;
  created_at: string;
}

export interface APIKey {
  id: string;
  org_id: string;
  name: string;
  prefix: string;
  created_by: string;
  created_at: string;
  last_used_at?: string;
  revoked_at?: string;
}

export interface Span {
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  service_name: string;
  operation_name: string;
  duration_ms: number;
  status_code: number;
  error_message?: string;
  tags?: Record<string, string>;
  start_time: string;
  start_offset_ms?: number;
}

export interface TraceRow {
  trace_id: string;
  root_service: string;
  root_operation: string;
  duration_ms: number;
  span_count: number;
  error_count: number;
  start_time: number;
}

export interface TraceDetail extends TraceRow {
  spans: Span[];
}

export interface ServiceNode {
  id: string;
  name: string;
  p99: number;
  health: 'healthy' | 'degraded' | 'critical';
  activeAnomalies?: number;
}

export interface ServiceEdge {
  source: string;
  target: string;
  rps: number;
  errorRate: number;
  p95Ms?: number;
  criticalPath?: boolean;
}

export interface ServiceGraph {
  nodes: ServiceNode[];
  edges: ServiceEdge[];
}

export interface ServiceStats {
  service_name: string;
  p50: number;
  p95: number;
  p99: number;
  error_rate: number;
  request_rate: number;
  from?: string;
  to?: string;
}

export interface Anomaly {
  id: string;
  org_id: string;
  service_name: string;
  operation_name: string;
  root_cause_service: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  z_score: number;
  baseline_latency_ms: number;
  observed_latency_ms: number;
  error_rate_delta: number;
  status: 'open' | 'investigating' | 'resolved';
  detected_at: string;
  resolved_at?: string;
  root_cause_path: string[];
}

export interface AnomalyStats {
  critical: number;
  high: number;
  medium: number;
  low: number;
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export function getAuthToken(): string | null {
  return localStorage.getItem('dt_token');
}

export function setAuthToken(token: string): void {
  localStorage.setItem('dt_token', token);
}

export function removeAuthToken(): void {
  localStorage.removeItem('dt_token');
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    let errMsg = `Request failed: ${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      if (body.error) {
        errMsg = body.error;
      }
    } catch {
      // ignore json parse error
    }
    throw new Error(errMsg);
  }

  return res.json() as Promise<T>;
}

export const authApi = {
  async register(email: string, password: string, organization: string) {
    const data = await request<{
      token: string;
      user: User;
      organization: Organization;
      api_key?: string;
    }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, organization }),
    });
    setAuthToken(data.token);
    return data;
  },

  async login(email: string, password: string) {
    const data = await request<{
      token: string;
      user: User;
      organization: Organization;
    }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    setAuthToken(data.token);
    return data;
  },

  async me() {
    return request<{ user: User; organization: Organization }>('/auth/me');
  },

  async listApiKeys() {
    return request<APIKey[]>('/auth/api-keys');
  },

  async createApiKey(name: string, prod: boolean) {
    return request<{ id: string; key: string; prefix: string; name: string; created_at: string }>('/auth/api-keys', {
      method: 'POST',
      body: JSON.stringify({ name, prod }),
    });
  },

  async revokeApiKey(id: string) {
    return request<{ status: string }>(`/auth/api-keys/${id}`, {
      method: 'DELETE',
    });
  },
};

export const tracesApi = {
  async search(params: {
    service?: string;
    status?: string;
    min_duration_ms?: number;
    max_duration_ms?: number;
    limit?: number;
    offset?: number;
  }) {
    const q = new URLSearchParams();
    if (params.service && params.service !== 'all') q.set('service', params.service);
    if (params.status && params.status !== 'all') q.set('status', params.status);
    if (params.min_duration_ms) q.set('min_duration_ms', String(params.min_duration_ms));
    if (params.max_duration_ms) q.set('max_duration_ms', String(params.max_duration_ms));
    if (params.limit) q.set('limit', String(params.limit));
    if (params.offset) q.set('offset', String(params.offset));

    const qs = q.toString();
    return request<TraceRow[]>(`/traces${qs ? `?${qs}` : ''}`);
  },

  async get(traceId: string) {
    return request<TraceDetail>(`/traces/${traceId}`);
  },
};

export const servicesApi = {
  async list() {
    return request<string[]>('/services');
  },

  async getGraph(from?: string, to?: string) {
    const q = new URLSearchParams();
    if (from) q.set('from', from);
    if (to) q.set('to', to);
    const qs = q.toString();
    return request<ServiceGraph>(`/services/graph${qs ? `?${qs}` : ''}`);
  },

  async getStats(service: string, from?: string, to?: string) {
    const q = new URLSearchParams();
    if (from) q.set('from', from);
    if (to) q.set('to', to);
    const qs = q.toString();
    return request<ServiceStats>(`/services/${encodeURIComponent(service)}/stats${qs ? `?${qs}` : ''}`);
  },
};

export interface ExplanationCost {
  model: string;
  input_tokens: number;
  output_tokens: number;
  estimated_cost_usd: number;
  cached: boolean;
  cost_ceiling_usd: number;
  hourly_cap_usd: number;
  hourly_spend_usd: number;
}

export interface IncidentExplanation {
  anomaly_id: string;
  org_id: string;
  root_cause_service: string;
  operation_name: string;
  confidence_score: number;
  summary: string;
  contributing_factors: string[];
  degraded_to_deterministic: boolean;
  fallback_reason?: string;
  cost_attribution: ExplanationCost;
  generated_at: string;
}

export const anomaliesApi = {
  async list(params?: { severity?: string; service?: string; status?: string; limit?: number; offset?: number }) {
    const q = new URLSearchParams();
    if (params?.severity && params.severity !== 'all') q.set('severity', params.severity);
    if (params?.service && params.service !== 'all') q.set('service', params.service);
    if (params?.status && params.status !== 'all') q.set('status', params.status);
    if (params?.limit) q.set('limit', String(params.limit));
    if (params?.offset) q.set('offset', String(params.offset));

    const qs = q.toString();
    return request<{ anomalies: Anomaly[]; stats: AnomalyStats }>(`/anomalies${qs ? `?${qs}` : ''}`);
  },

  async get(id: string) {
    return request<Anomaly>(`/anomalies/${id}`);
  },

  async updateStatus(id: string, status: 'open' | 'investigating' | 'resolved') {
    return request<{ id: string; status: string }>(`/anomalies/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
  },

  async explain(id: string, forceRefresh: boolean = false) {
    return request<IncidentExplanation>(`/anomalies/${id}/explain`, {
      method: 'POST',
      body: JSON.stringify({ force_refresh: forceRefresh }),
    });
  },
};

export const spansApi = {
  async ingest(spans: Partial<Span>[]) {
    return request<{ ingested: number; status: string }>('/spans', {
      method: 'POST',
      body: JSON.stringify(spans),
    });
  },
};

export interface CostSummary {
  orgId: string;
  infraShareUsd: number;
  llmSpendUsd: number;
  totalCostUsd: number;
  totalIncidentsExplained: number;
  avgCostPerIncidentUsd: number;
  hourlySpendUsd: number;
  hourlyCapUsd: number;
  activeOrgsCount: number;
}

export interface CostBreakdownItem {
  id: string;
  anomalyId: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  estimatedCostUsd: number;
  createdAt: string;
}

export const costsApi = {
  async getSummary() {
    return request<CostSummary>('/costs/summary');
  },
  async getBreakdown(params?: { limit?: number; offset?: number }) {
    const q = new URLSearchParams();
    if (params?.limit) q.set('limit', String(params.limit));
    if (params?.offset) q.set('offset', String(params.offset));
    const qs = q.toString();
    return request<{ items: CostBreakdownItem[]; limit: number; offset: number }>(`/costs/breakdown${qs ? `?${qs}` : ''}`);
  },
};

