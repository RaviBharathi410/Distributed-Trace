# Design Spec: Phase 5A — Root Cause Incident Explanation & Cost Attribution

## Fixed Context (§0–§2)
- **Audience:** Backend/Platform engineers evaluating self-hosted tracing and AIOps root-cause triage.
- **Non-negotiables:** No phantom claims. Real end-to-end execution. `org_id` tenant isolation mandatory across all queries and cost events. No unauthenticated endpoints. `main` always compiles. No LLM calls on the hot trace ingestion path. Flat, cost-effective pricing (§6 cost discipline).
- **Architectural Precedents:** Decision #26 (deterministic Welford + trace-tree critical-path traversal), Decision #29 (ClickHouse anomalies as single source of health truth), Decision #30 (Phase 5 scope split, on-demand AI analysis posture, pre-LLM cost dashboard mandate).

---

## 1. Feature Name & Problem Statement
- **Feature Name:** Deterministically-Grounded Root Cause Explanation (Phase 5A) & Tenant Cost Attribution.
- **Problem Statement:**
  When a latency or error anomaly strikes a distributed microservice topology, on-call engineers are alerted to the symptoms (e.g., checkout endpoint p99 spiked to 450ms) but must manually correlate the trace waterfall, database queries, and service dependencies to explain *why* the failure occurred. While Phase 3 solved the mathematical detection and root-cause identification deterministically ($O(1)$ Welford and critical-path traversal), raw metrics alone lack narrative context for rapid incident handoff.
  Furthermore, naive LLM integrations introduce runaway token expenses and operational hazards. This specification defines an on-demand, read-only incident explanation layer grounded strictly in Phase 3 deterministic outputs, constrained by a strict $\le \$0.01$/incident cost ceiling and validated by a mandatory tenant cost attribution dashboard stood up *before* any live LLM call is deployed.

---

## 2. Scope Boundaries & Non-Goals

### Included in Phase 5A (Immediate Scope):
1. **On-Demand Human-Readable Root Cause Explanation:** Formatted narrative diagnosis with confidence scoring, symptom summary, bottleneck service identification, and contributing factors.
2. **Deterministic Grounding & Zero-Hallucination Guardrails:** The LLM prompt is injected *only* with verified telemetry facts from Phase 3 (`root_cause_service`, `critical_path`, observed vs baseline latency, z-score, error rate delta). The LLM is never invoked on the ingestion path and cannot hallucinate services outside the trace tree.
3. **Idempotent Caching:** Explanation results are cached by incident ID and fingerprint. Re-opening or inspecting an active anomaly consumes $0 in LLM fees.
4. **Tenant Cost Attribution Infrastructure & Dashboard (§6/§7 Mandate):** Real-time tracking of token usage, model fees, and prorated infrastructure overhead per tenant (`org_id`), publishing the unit economics:
   $$\text{Tenant Monthly Cost} = \frac{\text{Infra Base Cost}}{\text{Active Orgs}} + \sum \text{Incident LLM Spend}$$

### Explicit Non-Goals (Deferred to Phase 5B / Separate Spec):
- **Autonomous Remediation & Infrastructure Mutation:** No automated execution of rollback scripts, kubectl commands, autoscaling adjustments, or configuration mutations against user systems.
- **Unsupervised Playbooks:** No automated background execution. Any future remediation proposal will require a dedicated safety threat model, blast radius isolation, and an explicit human-in-the-loop approval gate.
- **Automatic Ingestion-Time Invocation:** LLM calls are strictly user-triggered on-demand via the dashboard (`POST /api/v1/anomalies/{id}/explain`), never automatically dispatched per anomaly batch.

---

## 3. Model Tier, Budget & Cost Ceiling

| Parameter | Specification | Rationale |
|---|---|---|
| **Default Model Tier** | Google Gemini 1.5 Flash / Anthropic Claude 3.5 Haiku | Sub-second latency, high instruction fidelity for structured JSON extraction, ultra-low cost ($0.075 / 1M input tokens, $0.30 / 1M output tokens). |
| **Input Token Budget** | $\le 1,200$ tokens ($\sim 4.8\text{ KB}$ formatted telemetry context) | Bounded to the deterministic root-cause node, parent-child edge latencies, and top 3 contributing spans. |
| **Output Token Budget** | $\le 350$ tokens | Concise structured diagnosis, 3 bullet points of contributing factors, and confidence rating. |
| **Cost Per Invocation** | $\approx \$0.000195$ (Gemini 1.5 Flash) | $\approx 1/50\text{th}$ of one cent per incident explanation. |
| **Hard Cost Ceiling** | $\le \$0.01$ per incident | Enforced in code: requests rejecting if token estimate exceeds budget. |
| **Cache Lifetime** | 24 Hours (or until anomaly status changes) | Eliminates duplicate LLM invocations for shared on-call reviews. |

---

## 4. Tenant Cost Attribution & Dashboard Specification

Per §6 and §7 of the master architecture, the Cost-Per-Org dashboard is a **mandatory prerequisite** that must be operational before any LLM feature ships to production.

### Metrics & Storage Schema
A dedicated storage table records all LLM attribution events:
```sql
CREATE TABLE IF NOT EXISTS tenant_llm_costs (
    id UUID PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    anomaly_id VARCHAR(64) NOT NULL,
    model VARCHAR(64) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    estimated_cost_usd DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX idx_tenant_llm_costs_org ON tenant_llm_costs (org_id, created_at);
```

### Infrastructure Cost Proration
- **Infra Baseline:** Fixed at \$15.00/mo base hosting estimate (ClickHouse, Redpanda, Ingest Service).
- **Formula:**
  $$\text{Effective Org Cost} = \left(\frac{\$15.00}{\max(1, N_{\text{active\_orgs}})}\right) + \sum_{\text{billing period}} \text{estimated\_cost\_usd}$$

### Cost API Endpoints
1. `GET /api/v1/costs/summary`:
   - Returns current billing period totals: `infra_share_usd`, `llm_spend_usd`, `total_cost_usd`, `total_incidents_explained`, `avg_cost_per_incident_usd`.
2. `GET /api/v1/costs/breakdown`:
   - Returns per-incident cost ledger with timestamp, anomaly ID, model, token count, and cost in USD.

---

## 5. API Contract: Incident Explanation

### Endpoint
`POST /api/v1/anomalies/{id}/explain`

### Request Headers
- `Authorization: Bearer <jwt_or_api_key>`
- `Content-Type: application/json`
- `X-Request-ID: <uuid>`

### Request Body (Optional)
```json
{
  "force_refresh": false
}
```

### Response Schema (`200 OK`)
```json
{
  "anomaly_id": "anm_01J9...",
  "org_id": "org_enterprise_01",
  "root_cause_service": "payment-svc",
  "operation_name": "POST /charge",
  "confidence_score": 0.94,
  "summary": "Downstream database connection pool exhaustion in payment-svc caused p99 latency to spike from 15.2ms to 248.0ms (z-score: 5.4).",
  "contributing_factors": [
    "PostgreSQL connection acquisition wait time increased by 210ms on 'payment-svc'",
    "Cascading latency propagated upstream to 'checkout-api' (edge latency p95: 260ms)",
    "Upstream error rate remained under 1%, indicating request queuing rather than circuit breaking"
  ],
  "recommended_investigation": [
    "Inspect database connection pool metrics (active vs idle) on payment-db",
    "Verify slow query logs for lock contention on the 'charges' table"
  ],
  "cost_attribution": {
    "model": "gemini-1.5-flash",
    "input_tokens": 842,
    "output_tokens": 184,
    "estimated_cost_usd": 0.000118,
    "cached": false,
    "cost_ceiling_usd": 0.010000
  },
  "generated_at": "2026-10-01T18:30:00Z"
}
```

### Error Responses
- `401 Unauthorized`: Missing or invalid tenant credentials.
- `403 Forbidden`: User org does not match anomaly `org_id` (strict tenant isolation).
- `404 Not Found`: Anomaly ID not found for tenant.
- `429 Too Many Requests`: Anomaly cost ceiling exceeded for organization billing cycle.
- `503 Service Unavailable`: LLM provider unreachable (falls back gracefully to deterministic telemetry summary).

---

## 6. Acceptance Criteria (Numbered & Testable)

1. **Prerequisite Dashboard Verification:** `GET /api/v1/costs/summary` returns valid JSON with `infra_share_usd`, `llm_spend_usd`, and `avg_cost_per_incident_usd` strictly isolated to the caller's `org_id`.
2. **Tenant Isolation on Cost Attribution:** Anomaly explanation requests under Org A record cost events visible ONLY to Org A; Org B cost summaries remain \$0.00.
3. **Deterministic Telemetry Grounding:** Prompt generation strictly uses the anomaly's deterministic fields (`root_cause_service`, `z_score`, `baseline_latency_ms`, `observed_latency_ms`). If the service name is not in the trace graph, the generator rejects prompt generation.
4. **Per-Incident Cost Ceiling Enforcement:** Code validates that estimated prompt token size satisfies $\text{estimated\_cost} \le \$0.01$. Any payload exceeding the token budget is rejected before issuing an outbound LLM request.
5. **Idempotent Cache Behavior:** Consecutive calls to `POST /api/v1/anomalies/{id}/explain` return the cached explanation with `cost_attribution.cached == true` and \$0.00 incremental cost.
6. **Zero Live Mutation:** The explanation engine has no access to infrastructure APIs (no Kubernetes client, no AWS/GCP SDK, no shell exec). Output is pure read-only JSON text.
7. **CI Pass & Zero Credential Leakage:** All unit and integration tests compile and pass on GitHub Actions with real ClickHouse and Redpanda service containers. Zero API keys committed or logged.
