# Design Phase — Design Spec Template

## Fixed Context (§0–§2)
- **Audience:** Backend/Platform engineers evaluating self-hosted tracing.
- **Non-negotiables:** No phantom claims. Real end-to-end execution. `org_id` tenant isolation mandatory. No unauthenticated endpoints. `main` always compiles. No LLM calls on hot path. Flat, cost-effective pricing.

---

## Design Spec Structure
- **Feature Name & Problem Statement:** What problem does this solve, and why now?
- **Non-Goals:** What is explicitly out of scope for this increment?
- **API Contract:**
  - Method & Path: `METHOD /api/v1/...`
  - Headers: `Authorization: Bearer <token>`, `X-Request-ID: <uuid>`
  - Request Schema (JSON)
  - Response Schema (JSON success and error codes)
- **Data Model Changes:**
  - PostgreSQL / ClickHouse migration DDL
  - Multi-tenant column (`org_id`) & indexes
- **Acceptance Criteria (Numbered & Testable):**
  1. ...
  2. ...
- **Open Questions:**
