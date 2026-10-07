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
  Furthermore, naive LLM integrations introduce runaway token expenses and operational hazards. This specification defines an on-demand, read-only incident explanation layer grounded strictly in Phase 3 deterministic outputs, bounded by an **enforced** $\le \$0.01$/incident cost ceiling, and validated by a mandatory tenant cost attribution dashboard stood up *before* any live LLM call is deployed.

---

## 2. Scope Boundaries & Non-Goals

### Included in Phase 5A (Immediate Scope):
1. **On-Demand Human-Readable Root Cause Explanation:** Formatted narrative diagnosis with confidence scoring, symptom summary, bottleneck service identification, and contributing factors.
2. **Deterministic Grounding (Input Guardrail):** The LLM prompt is injected *only* with verified telemetry facts from Phase 3 (`root_cause_service`, `critical_path`, observed vs baseline latency, z-score, error rate delta). The LLM is never invoked on the ingestion path and cannot hallucinate services outside the trace tree.
3. **Output-Side Containment (Prompt & Schema Guardrail):** The system prompt explicitly forbids the model from volunteering remediation suggestions, mitigation steps, rollback instructions, shell commands, or "you should also check X" infrastructure directives. Output is strictly confined to descriptive diagnosis.
4. **Enforced Cost Circuit Breaker:** The $\le \$0.01$/incident ceiling is enforced *before the fact* via a pre-flight token-cost gate, supplemented by a tenant-level hourly spend ceiling ($\le \$1.00/\text{hr}$) to prevent cascading failure storms from exhausting budgets.
5. **Idempotent Caching:** Explanation results are cached by incident ID and fingerprint for 24 hours. Re-opening or inspecting an active anomaly consumes $0 in LLM fees.
6. **Tenant Cost Attribution Infrastructure & Dashboard (§6/§7 Mandate):** Real-time tracking of token usage, model fees, and prorated infrastructure overhead per tenant (`org_id`), publishing the unit economics:
   $$\text{Tenant Monthly Cost} = \frac{\text{Infra Base Cost (Calibrated)}}{\text{Active Orgs}} + \sum \text{Incident LLM Spend}$$

### Explicit Non-Goals (Deferred to Phase 5B / Separate Spec):
- **Autonomous Remediation & Infrastructure Mutation:** No automated execution of rollback scripts, kubectl commands, autoscaling adjustments, or configuration mutations against user systems.
- **Unsupervised Playbooks:** No automated background execution. Any future remediation proposal will require a dedicated safety threat model, blast radius isolation, and an explicit human-in-the-loop approval gate.
- **Automatic Ingestion-Time Invocation:** LLM calls are strictly user-triggered on-demand via the dashboard (`POST /api/v1/anomalies/{id}/explain`), never automatically dispatched per anomaly batch.

---

## 3. Model Tier, Budget & Enforced Cost Ceilings

### Planning Projections vs Empirical Telemetry
> [!IMPORTANT]
> The unit economics and base costs listed below are initial **planning projections**, not observed production measurements. No LLM calls have shipped yet. The explicit objective of standing up the Cost-Per-Org dashboard in Step 1 is to replace these initial planning assumptions with empirical telemetry (actual token counts, measured API latencies, and actual container compute footprints) before declaring the $\$0.01$ ceiling validated.

| Parameter | Specification | Classification | Rationale |
|---|---|---|---|
| **Default Model Tier** | Google Gemini 1.5 Flash / Anthropic Claude 3.5 Haiku | Configuration | Sub-second latency, high instruction fidelity for structured JSON extraction, ultra-low cost ($0.075 / 1M input tokens, $0.30 / 1M output tokens). |
| **Input Token Budget** | $\le 1,200$ tokens ($\sim 4.8\text{ KB}$ formatted telemetry context) | Enforced Limit | Bounded to the deterministic root-cause node, parent-child edge latencies, and top 3 contributing spans. |
| **Output Token Budget** | $\le 350$ tokens | Enforced Limit | Concise structured diagnosis, 3 bullet points of contributing factors, and confidence rating. |
| **Projected Cost Per Call** | $\approx \$0.000195$ (Gemini 1.5 Flash) | **Planning Estimate** | Calculated from published rates: $(1200 \times 0.075 + 350 \times 0.30) / 10^6 \approx \$0.000195$. Subject to empirical calibration via dashboard. |
| **Per-Incident Cost Ceiling** | $\le \$0.01$ per incident | **Enforced Limit** | Pre-flight circuit breaker: rejects requests if pre-calculated token cost exceeds \$0.01 before invoking the LLM provider. |
| **Tenant Storm Spend Cap** | $\le \$1.00$ / hour per organization | **Enforced Limit** | Short-circuits cascading failure storms (operator mashing or storm alerts) by returning `429` with deterministic $0 fallback. |
| **Cache Lifetime** | 24 Hours (or until anomaly status changes) | Configuration | Eliminates duplicate LLM invocations for shared on-call reviews. |

---

## 4. Tenant Cost Attribution & Dashboard Specification

Per §6 and §7 of the master architecture, the Cost-Per-Org dashboard is a **mandatory prerequisite** that must be operational before any LLM feature ships to production.

### Metrics & Storage Schema
A dedicated PostgreSQL table records all LLM attribution events:
```sql
CREATE TABLE IF NOT EXISTS tenant_llm_costs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id VARCHAR(64) NOT NULL,
    anomaly_id VARCHAR(64) NOT NULL,
    model VARCHAR(64) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    estimated_cost_usd DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_llm_costs_org ON tenant_llm_costs (org_id, created_at);
```

### Infrastructure Cost Proration
- **Infra Planning Baseline:** Nominal \$15.00/mo base hosting estimate (ClickHouse, Redpanda, Ingest Service).
- **Dashboard Calibration Role:** Step 1 exposes this figure as a configurable base, displaying whether tenant share + LLM spend tracks within the target margins.
- **Formula:**
  $$\text{Effective Org Cost} = \left(\frac{\$15.00\text{ (Projected Baseline)}}{\max(1, N_{\text{active\_orgs}})}\right) + \sum_{\text{billing period}} \text{estimated\_cost\_usd}$$

### Cost API Endpoints
1. `GET /api/v1/costs/summary`:
   - Returns current billing period totals: `infra_share_usd`, `llm_spend_usd`, `total_cost_usd`, `total_incidents_explained`, `avg_cost_per_incident_usd`, `hourly_spend_usd`, `hourly_cap_usd`.
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
  "summary": "Downstream database connection acquisition delay in payment-svc caused p99 latency to spike from 15.2ms to 248.0ms (z-score: 5.4).",
  "contributing_factors": [
    "Database connection wait time increased by 210ms on 'payment-svc'",
    "Cascading latency propagated upstream to 'checkout-api' (edge latency p95: 260ms)",
    "Upstream error rate remained under 1%, indicating request queuing rather than hard circuit breaking"
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
- `429 Too Many Requests`: Per-incident cost ceiling exceeded or tenant hourly spend cap reached (\$1.00/hr). Returns deterministic fallback.
- `503 Service Unavailable`: LLM provider unreachable (falls back gracefully to deterministic telemetry summary).

---

## 6. Output-Side Guardrails & System Prompt Directives

To strictly enforce the Phase 5A / 5B boundary, output-side safety is governed by a two-layer defense: contrastive prompt guidance and a deterministic regex-based advisory detector.

### System Prompt Directives & Contrastive Few-Shot Guidance
```text
You are a read-only root-cause diagnostic engine for distributed traces.
Your SOLE task is to explain what occurred based strictly on the provided telemetry facts.

CRITICAL SAFETY & SCOPE RULES:
1. Do NOT volunteer or suggest remediation actions, operational fixes, or configuration changes.
2. Do NOT use soft advisory phrasing (e.g., "this pattern is often resolved by...", "typically indicates the service needs more replicas", "consider tuning...").
3. Do NOT generate shell commands, kubectl directives, or infrastructure mutation instructions.
4. Confine your response strictly to descriptive diagnosis of observed telemetry anomalies and contributing factors.
5. Output must be valid JSON matching the exact schema provided.

CONTRASTIVE EXAMPLES:

❌ VIOLATION (Soft Advisory / Prescriptive Phrasing - FORBIDDEN):
{
  "summary": "This pattern is often resolved by increasing connection pool size on payment-svc.",
  "contributing_factors": [
    "Typically indicates the service needs more replicas to handle traffic bursts.",
    "This is commonly addressed by a cache warm restart."
  ]
}

✅ COMPLIANT (Descriptive Observation of Telemetry Only - REQUIRED):
{
  "summary": "Database connection acquisition wait time in 'payment-svc' spiked from 15ms to 240ms, causing request queue buildup.",
  "contributing_factors": [
    "Active connections reached the configured pool limit of 100 at 14:02:10 UTC.",
    "Upstream edge 'checkout-api' -> 'payment-svc' observed 260ms p95 latency while error rate remained at 0.4%."
  ]
}
```

### Two-Tiered Response Validation Engine
A post-generation deterministic validator scans the output before it is returned to the user:
1. **Tier 1 (Literal Command / Tooling Blocklist):**
   - Matches literal infrastructure mutation keywords: `kubectl`, `helm`, `rollback`, `reboot`, `scale up`, `scale down`, `deploy`, `apply`.
2. **Tier 2 (Soft Advisory Pattern Detector):**
   - Regex-based matching on prescriptive and advisory phrasing constructs that attempt to volunteer fixes without using blocked keywords:
     - `\b(resolved by|addressed by|fixed by|remedied by|mitigated by|workaround is)\b`
     - `\b(should (increase|decrease|add|scale|restart|configure|tune|upgrade|revert|change|deploy|apply|check))\b`
     - `\b(needs (more|fewer|additional|to be|scaling|tuning))\b`
     - `\b(recommend(ed)? (increasing|decreasing|adding|scaling|restarting|configuring|to))\b`
     - `\b(consider (increasing|decreasing|adding|scaling|restarting|tuning))\b`
     - `\b(try (restarting|increasing|scaling|reverting))\b`
     - `\b(best practice is to|solution is to|next step is to)\b`
3. **Validator Rejection Fallback Policy (Hard Stop — Zero Re-Prompts):**
   - If either Tier 1 (commands) or Tier 2 (soft advisories) matches, the engine **does NOT re-prompt the LLM**. Issuing a second call would compound token consumption and risk exceeding the \$0.01 per-incident ceiling.
   - The engine immediately degrades to a **100% deterministic Phase 3 telemetry summary** ($0 incremental cost, 0 operational risk).
   - The response payload sets `"degraded_to_deterministic": true` and `"fallback_reason": "Output contained forbidden advisory/remediation phrasing; degraded to deterministic telemetry observation."`
   - The single LLM call's token usage is recorded in `tenant_llm_costs` for honest billing transparency, but no further API calls are made.

### Centralized Pricing & Vendor Drift Prevention
- All model pricing parameters and ceiling constants are maintained in application configuration (`internal/config/config.go` and `.env.example`), avoiding hardcoded pricing numbers:
  - `LLM_MODEL_NAME` (default: `"gemini-1.5-flash"`)
  - `LLM_INPUT_PRICE_PER_MILLION` (default: `$0.075 / 1M` tokens, effective Oct 2024)
  - `LLM_OUTPUT_PRICE_PER_MILLION` (default: `$0.300 / 1M` tokens, effective Oct 2024)
  - `LLM_COST_CEILING_PER_INCIDENT` (default: `$0.010` max per incident)
  - `LLM_HOURLY_SPEND_CAP` (default: `$1.000` max per tenant org)
- Vendor pricing changes are managed via environment variables and tracked in the architectural decision log without requiring codebase refactoring.

### Frontend Treatment for Degraded Fallback State (§5 / §6 Honesty Standard)
To prevent misleading operators into believing a fallback was generated by an LLM (violating the core honesty and transparency standard), the frontend drawer explicitly discriminates between three visual states:
1. **Unanalyzed State (Pre-Invocation):**
   - Shows action button: `[⚡ Explain Root Cause (≤ $0.01)]` with subtext *"Grounded in Phase 3 telemetry · Subject to output safety validator & $1.00/hr storm spend cap"*.
2. **AI Prose Diagnostic State (`degraded_to_deterministic == false`):**
   - Purple/indigo accent header: `✦ AI Root Cause Diagnosis (gemini-1.5-flash)`.
   - Confidence score indicator (e.g., `94% Confidence`).
   - Cost attribution pill: `${cost_attribution.estimated_cost_usd} · ${total_tokens} tokens` (or `Cached ($0.00)`).
   - Generative prose summary and bulleted contributing factors.
3. **Degraded Fallback State (`degraded_to_deterministic == true`):**
   - Amber/slate accent header: `⚡ Fallback: Deterministic Telemetry Diagnosis`.
   - **Prominent Transparency Banner:**
     ```
     AI explanation unavailable / bypassed: {fallback_reason}
     Displaying verified Phase 3 mathematical telemetry findings ($0 LLM fee incurred).
     ```
   - **Mathematical Telemetry Card Grid:** Renders raw, verified Phase 3 telemetry directly:
     - Root-cause bottleneck service and endpoint operation.
     - Observed latency vs historical baseline and Bessel-corrected $Z$-score.
     - Critical-path graph traversal edge identifying the downstream blocker.
   - **Cost Pill:** Displays `$0.000000 LLM Incurred` (or if degraded post-validation, displays the actual recorded tokens labeled `Rejected by Output Validator — Logged to Ledger`).

---

## 7. Acceptance Criteria (Numbered & Testable)

1. **Prerequisite Dashboard Verification:** `GET /api/v1/costs/summary` returns valid JSON with `infra_share_usd`, `llm_spend_usd`, and `avg_cost_per_incident_usd` strictly isolated to the caller's `org_id` before any LLM call is wired.
2. **Tenant Isolation on Cost Attribution:** Anomaly explanation requests under Org A record cost events visible ONLY to Org A; Org B cost summaries remain \$0.00.
3. **Deterministic Telemetry Grounding:** Prompt generation strictly uses the anomaly's deterministic fields (`root_cause_service`, `z_score`, `baseline_latency_ms`, `observed_latency_ms`). If the service name is not in the trace graph, prompt generation is aborted.
4. **Pre-Flight Cost Ceiling Enforcement:** Code calculates estimated prompt token cost before dispatching outbound HTTP calls. Any payload with estimated cost $> \$0.01$ is rejected pre-flight.
5. **Tenant Storm Circuit Breaker:** Consecutive explanation requests exceeding the \$1.00/hr org spend cap are short-circuited with `429` and served with a \$0 deterministic fallback.
6. **Output-Side Remediation & Soft-Advisory Ban Assertion:** Dedicated unit tests verify that the validator rejects:
   - Direct imperative commands (`kubectl scale`, `rollback`).
   - Soft advisory phrasing (`this pattern is often resolved by increasing connection pool size`, `typically indicates the service needs more replicas`, `this is commonly addressed by a cache warm restart`).
   - Pure descriptive telemetry statements pass validation cleanly.
7. **Idempotent Cache Behavior:** Consecutive calls to `POST /api/v1/anomalies/{id}/explain` return the cached explanation with `cost_attribution.cached == true` and \$0.00 incremental cost.
8. **Zero Live Mutation:** The explanation engine has no access to infrastructure APIs (no Kubernetes client, no AWS/GCP SDK, no shell exec). Output is pure read-only JSON text.
9. **CI Pass & Zero Credential Leakage:** All unit and integration tests compile and pass on GitHub Actions with real ClickHouse and Redpanda service containers. Zero API keys committed or logged.

