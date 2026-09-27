# Discovery Phase — Research Brief Template

## Fixed Context (§0–§2)
- **Audience:** Backend/Platform engineers evaluating self-hosted tracing.
- **Non-negotiables:** No phantom claims. Real end-to-end execution. `org_id` tenant isolation mandatory. No unauthenticated endpoints. `main` always compiles. No LLM calls on hot path. Flat, cost-effective pricing.

---

## Research Brief Structure
- **Question:** What specific question needs resolution?
- **Standard Questions (§3.1):**
  1. What does the code actually do today (file + line citations)?
  2. Who is the user of this feature, and what breaks if we ship nothing?
  3. What is the smallest change that unblocks reality vs. a nice-to-have?
  4. What existing component/table/endpoint could this reuse?
  5. What is the rollback plan?
- **Method:** Exact files viewed and CLI commands executed.
- **Finding:** Specific observable behavior and code evidence.
- **Confidence:** High / Medium / Low.
- **Implication:** Direct impact on architecture, cost, and next phase.
