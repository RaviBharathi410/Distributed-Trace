# Parameterized Scenario Prompts (§5)

## 1. Competitor Analysis
> "Compare DistributedTrace's actual current capabilities (not roadmap claims) against [Jaeger / Tempo / Honeycomb] on: ingestion protocol support, query latency at scale, multi-tenancy, pricing model. Flag any capability we claim publicly that we cannot currently demonstrate."

## 2. Feature Prioritization
> "Given the open backlog, score each item on: (a) does it unblock the compiling/connected baseline, (b) user-requested vs. assumed, (c) engineering cost in days, (d) security/tenant risk if skipped. Rank and justify the top 5."

## 3. Roadmap Planning
> "Propose a phase sequence from current state (40% complete, backend non-functional) to a demo-able MVP. Each phase must state its exit criteria as a runnable command or observable behavior, not a percentage."

## 4. Technical Debt Assessment
> "List every instance where documentation, UI copy, or comments claim functionality that the code does not implement. For each, decide: build it, or correct the claim — never leave the discrepancy standing."

## 5. QA/Testing Plan
> "Given zero existing tests, propose a minimal test plan that covers: auth middleware rejection of bad tokens, tenant isolation (org A cannot query org B's spans), ingestion validation rejecting malformed spans, and the trace-search API's pagination. Prioritize tests that would have caught the current compilation failures."

## 6. Deployment Considerations
> "We have no Dockerfile, no docker-compose, no CI. Propose the minimum viable local dev environment (Postgres + ClickHouse + Kafka) and a CI pipeline that fails the build the same way local compilation currently fails, so this class of breakage can't reach main again."
