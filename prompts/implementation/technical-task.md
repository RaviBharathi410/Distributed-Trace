# Implementation Phase — Technical Task & QA Template

## Fixed Context (§0–§2)
- **Audience:** Backend/Platform engineers evaluating self-hosted tracing.
- **Non-negotiables:** No phantom claims. Real end-to-end execution. `org_id` tenant isolation mandatory. No unauthenticated endpoints. `main` always compiles. No LLM calls on hot path. Flat, cost-effective pricing.

---

## Technical Task Structure
- **Title:** Descriptive title
- **Files Touched:** Clickable links to all modified files
- **Definition of Done:** Clear conditions for completion
- **Test Requirement:** Exact unit/integration test assertions
- **Verification Commands:** `go test -v ./...`, `go build ./...`, `npm run build`
- **Estimated Effort:** Hours/Days
