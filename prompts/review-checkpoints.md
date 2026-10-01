# Review Checkpoints

## Checkpoint: 2026-09-27 — Initial Baseline & Audit Adoption
- **Status:** Baseline established from Technical Completion Status Audit.
- **Project Completion:** ~40.5% (High confidence).
- **What Changed Since Last Checkpoint:**
  - Adopted Master Prompt and Phased Plan (§0–§8).
  - Confirmed working assumptions: backend/platform engineering audience, phase-gated cadence, non-negotiable end-to-end reality check, mandatory `org_id` isolation, compile-clean `main`, no hot-path LLM calls.
  - Initialized `/prompts/` repository structure, `decision-log.md`, and checkpoint tracker.
- **Next Checkpoint's Exit Criteria (Phase 0 Exit):**
  1. All Go compiler errors resolved across all packages.
  2. `cmd/server/main.go` entrypoint implemented with config, logging, metrics, and healthcheck router.
  3. `docker-compose.yml` operational for Postgres, ClickHouse, and Redpanda.
  4. `.env.example` created with working local defaults.
  5. `go build ./...` exits with code 0.

---

## Checkpoint: 2026-09-27 — Phase 0 Complete: Compilation & Environment Unblocked
- **Status:** ✅ **Phase 0 Complete — All Exit Criteria Satisfied**.
- **Project Completion:** ~52% (Backend compilable + entrypoint + local infra scaffolding).
- **What Changed Since Last Checkpoint:**
  1. **All Go Compiler Errors Fixed:**
     - Removed duplicate `HashPassword` by removing [`internal/auth/hash.go`](file:///d:/Projects/DistributedTrace/internal/auth/hash.go) and consolidating into [`internal/auth/password.go`](file:///d:/Projects/DistributedTrace/internal/auth/password.go).
     - Fixed unused imports in [`internal/config/config.go`](file:///d:/Projects/DistributedTrace/internal/config/config.go), [`internal/auth/apikey.go`](file:///d:/Projects/DistributedTrace/internal/auth/apikey.go), [`internal/observability/logger.go`](file:///d:/Projects/DistributedTrace/internal/observability/logger.go), and [`internal/observability/telemetry.go`](file:///d:/Projects/DistributedTrace/internal/observability/telemetry.go).
     - Fixed `internal/ingest/producer.go`: imported `prometheus` and switched to standard `prometheus.NewTimer()`.
     - Wrapped `DbQueryDuration` in [`internal/observability/metrics.go`](file:///d:/Projects/DistributedTrace/internal/observability/metrics.go) with `QueryDurationTracker` returning `*prometheus.Timer` on `WithLabelValues(...)`, solving 32 timer compilation errors across Postgres and ClickHouse repositories.
     - Resolved JWT RS256 asymmetric signing with fallback ephemeral key generation in [`internal/auth/jwt.go`](file:///d:/Projects/DistributedTrace/internal/auth/jwt.go) per §2.
  2. **Created Server Entrypoint:**
     - Built [`cmd/server/main.go`](file:///d:/Projects/DistributedTrace/cmd/server/main.go) with Chi router, request ID, CORS, logger/metric middleware, `/healthz`, `/metrics`, and graceful signal shutdown.
  3. **Multi-Tenant Isolation (Non-Negotiable #2):**
     - Updated ClickHouse [`migrations/clickhouse/001_create_otel_spans.sql`](file:///d:/Projects/DistributedTrace/migrations/clickhouse/001_create_otel_spans.sql) to include `org_id String` with partition/order index.
     - Updated [`internal/repository/clickhouse/traces.go`](file:///d:/Projects/DistributedTrace/internal/repository/clickhouse/traces.go) to append `s.OrgID` and enforce `WHERE org_id = ?` in search and trace retrieval.
  4. **Local Infrastructure & Dev Environment:**
     - Created root [`docker-compose.yml`](file:///d:/Projects/DistributedTrace/docker-compose.yml) providing Postgres 16, ClickHouse 24.3, and Redpanda (Kafka-compatible).
     - Created [`.env.example`](file:///d:/Projects/DistributedTrace/.env.example) with pre-configured local connection strings.
  5. **Tooling & Frontend Build:**
     - Replaced broken ESLint 9 configuration with [`.eslintrc.cjs`](file:///d:/Projects/DistributedTrace/.eslintrc.cjs) matching ESLint 8.57, running with 0 errors.
     - Verified `npm run build` succeeds (`dist/` generated in 4s).
- **Verification Evidence (Commands Run):**
  - `go build ./...` -> Exited 0 (No compilation errors).
  - `go build -o distributedtrace-server.exe ./cmd/server` -> Exited 0.
  - `npx eslint . --ext ts,tsx` -> Exited 0 (0 errors, 10 warnings).
  - `npm run build` -> Exited 0.
- **Next Checkpoint's Exit Criteria (Phase 1 — MVP Core REST API & Connected Frontend):**
  1. Core REST API endpoints implemented and mounted in Chi:
     - Auth: `/api/v1/auth/login`, `/api/v1/auth/register`, `/api/v1/auth/api-keys`.
     - Traces: `/api/v1/traces` (search), `/api/v1/traces/{id}` (waterfall).
     - Services: `/api/v1/services`, `/api/v1/services/graph`.
     - Anomalies: `/api/v1/anomalies`, `/api/v1/anomalies/{id}`.
  2. Auth middleware (`RequireAuth`, `RequireAPIKey`, `ExtractOrgID`) applied to all protected routes.
  3. Frontend dashboard tabs ([`TracesTab.tsx`](file:///d:/Projects/DistributedTrace/src/components/dashboard/TracesTab.tsx), [`ServiceMapTab.tsx`](file:///d:/Projects/DistributedTrace/src/components/dashboard/ServiceMapTab.tsx), [`AnomaliesTab.tsx`](file:///d:/Projects/DistributedTrace/src/components/dashboard/AnomaliesTab.tsx)) wired to `@tanstack/react-query` consuming live endpoints.
  4. User can log in, ingest a span, and observe it in the UI with zero mocked data.

---

## Checkpoint: 2026-09-27 — Phase 1 Complete: Core REST API & Connected Frontend
- **Status:** ✅ **Phase 1 Complete — Handlers Implemented, Auth Gated, Frontend Wired, Dead Code Removed**.
- **Project Completion:** ~68% (Full control plane + data-layer API + connected React Query frontend).
- **What Changed Since Last Checkpoint:**
  1. **Addressed Pre-Phase 1 Hardening:**
     - Confirmed ephemeral RSA key generation in [`internal/auth/jwt.go`](file:///d:/Projects/DistributedTrace/internal/auth/jwt.go) is strictly gated by `env == "development" || env == "test"`, failing fast in production.
     - Resolved all 10 ESLint warnings (`npx eslint . --ext ts,tsx` clean with 0 warnings, 0 errors).
     - Decided and logged dual-mode ingestion auth (`POST /api/v1/spans`): accepts either `X-API-Key` (service-to-service) or user JWT `Bearer <token>` (dashboard / browser / test runs), both resolving and injecting `org_id` into context.
  2. **Auth & Ingestion Middleware:**
     - Created [`internal/auth/middleware.go`](file:///d:/Projects/DistributedTrace/internal/auth/middleware.go): `RequireUserAuth`, `RequireIngestAuth`, and `RequireRole`.
  3. **PostgreSQL & ClickHouse API Controllers:**
     - [`internal/api/response.go`](file:///d:/Projects/DistributedTrace/internal/api/response.go): `WriteJSON`, `WriteError`.
     - [`internal/api/auth.go`](file:///d:/Projects/DistributedTrace/internal/api/auth.go): `Register`, `Login`, `Me`, `ListAPIKeys`, `CreateAPIKey`, `RevokeAPIKey`.
     - [`internal/api/traces.go`](file:///d:/Projects/DistributedTrace/internal/api/traces.go): `SearchTraces`, `GetTraceByID` (trace with waterfall spans).
     - [`internal/api/services.go`](file:///d:/Projects/DistributedTrace/internal/api/services.go): `ListServices`, `GetServiceGraph`, `GetServiceStats`.
     - [`internal/api/anomalies.go`](file:///d:/Projects/DistributedTrace/internal/api/anomalies.go): `ListAnomalies`, `GetAnomalyByID`, `UpdateAnomalyStatus`.
     - [`internal/api/spans.go`](file:///d:/Projects/DistributedTrace/internal/api/spans.go): `IngestSpans` (validates, enriches with tenant `org_id`, persists to ClickHouse, and optionally forwards to Kafka).
     - [`internal/repository/postgres/orgs.go`](file:///d:/Projects/DistributedTrace/internal/repository/postgres/orgs.go): Organization persistence repository.
  4. **Server Routing & DB Wiring:**
     - Mounted all API controllers and database pools into [`cmd/server/main.go`](file:///d:/Projects/DistributedTrace/cmd/server/main.go) under `/api/v1`.
  5. **Dead Wireframe Code Purged:**
     - Deleted dead mockup pages: `src/pages/Dashboard/Dashboard.tsx`, `src/pages/Landing/LandingPage.tsx`, and `src/components/common/AppShell.tsx`.
  7. **Complete Read-Path Tenant Isolation & Hardening:**
     - Enforced `org_id` exclusively from auth context across all Service handlers ([`internal/api/services.go`](file:///d:/Projects/DistributedTrace/internal/api/services.go)) and ClickHouse Service repository queries ([`internal/repository/clickhouse/services.go`](file:///d:/Projects/DistributedTrace/internal/repository/clickhouse/services.go)), eliminating cross-tenant leakage in `ListServices`, `GetServiceGraph`, and `GetServiceStats`.
     - Confirmed zero read handlers accept `org_id` from client query params or request bodies.
  8. **ClickHouse Multi-Tiered Retention Policy:**
     - Updated [`migrations/clickhouse/001_create_otel_spans.sql`](file:///d:/Projects/DistributedTrace/migrations/clickhouse/001_create_otel_spans.sql) with conditional TTL: 14 days for successful spans (`status_code != 2`), 90 days for error spans.
     - Added 90-day TTL to [`migrations/clickhouse/002_create_anomalies.sql`](file:///d:/Projects/DistributedTrace/migrations/clickhouse/002_create_anomalies.sql).
     - Added 180-day TTL to [`migrations/clickhouse/003_create_service_metrics_hourly.sql`](file:///d:/Projects/DistributedTrace/migrations/clickhouse/003_create_service_metrics_hourly.sql).
  9. **RBAC & CORS Gating Mounted:**
     - Mounted [`RequireRole(domain.RoleAdmin)`](file:///d:/Projects/DistributedTrace/internal/auth/middleware.go#L138) on sensitive API key creation and revocation endpoints in [`cmd/server/main.go`](file:///d:/Projects/DistributedTrace/cmd/server/main.go).
     - Mounted [`RequireRole(domain.RoleMember, domain.RoleAdmin)`](file:///d:/Projects/DistributedTrace/internal/auth/middleware.go#L138) on anomaly status update endpoint.
     - Hardened CORS middleware with an explicit allowlist checking against configured `AllowedOrigin` and local development origins, rejecting wildcard `*` with credentials.
   10. **Automated CI Workflow with Native ClickHouse Service Container (Verified Green):**
      - Added [`.github/workflows/ci.yml`](file:///d:/Projects/DistributedTrace/.github/workflows/ci.yml) protecting `main` on push and PR.
      - Includes a native ClickHouse service container (`clickhouse/clickhouse-server:24.3-alpine`) with explicit environment credentials (`CLICKHOUSE_USER: default`, `CLICKHOUSE_PASSWORD: ""`, `CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1`), and IPv4 healthcheck (`http://127.0.0.1:8123/ping`).
      - Verified green on GitHub Actions: **Run ID [36327125337](https://github.com/RaviBharathi410/Distributed-Trace/actions/runs/36327125337)** completed with status **`success`**, with `TestGetServiceGraph_RealClickHouse_Integration` executing and passing in **0.02s** on the live CI service container.
   11. **Live ClickHouse Join-Path Integration Test (Canonical Migration DDL, Zero Drift):**
      - Implemented [`internal/repository/clickhouse/integration_test.go`](file:///d:/Projects/DistributedTrace/internal/repository/clickhouse/integration_test.go) (`TestGetServiceGraph_RealClickHouse_Integration`).
      - Replaced inline DDL with dynamic loading and execution of [`migrations/clickhouse/001_create_otel_spans.sql`](file:///d:/Projects/DistributedTrace/migrations/clickhouse/001_create_otel_spans.sql) as the single source of truth.
      - Connected to live ClickHouse container (`dt-clickhouse`), created MergeTree table with partitioning, bloom filter indexes, and multi-tiered TTL, inserted cross-tenant parent-child spans for Org A and Org B, and executed the real ClickHouse `JOIN` query in `GetServiceGraph`.
      - Proved on live ClickHouse MergeTree engine that zero Org B nodes/edges appear in Org A's graph, and Org A's edge is cleanly assembled.
      - Fixed ClickHouse 24.3 TTL engine syntax: `DateTime64` requires `toDateTime(start_time)` in TTL clauses (resolved DB Exception 450).
   12. **RoleOwner Superuser Bypass Explicitly Documented:**
      - Confirmed and documented in [`prompts/decision-log.md`](file:///d:/Projects/DistributedTrace/prompts/decision-log.md) (Decision #15). `RequireRole` intentionally permits `RoleOwner` so organization creators are never locked out of administrative or operational actions.
- **Verification Evidence (Commands Run):**
  - **GitHub Actions Remote CI Run:** [Run 36327125337](https://github.com/RaviBharathi410/Distributed-Trace/actions/runs/36327125337) -> **COMPLETED SUCCESS** (Both `Backend` and `Frontend` jobs green, live ClickHouse integration test passed in 0.02s).
  - `go test -v -count=1 ./internal/repository/clickhouse/... -run RealClickHouse` -> Exited 0 (**PASS: 0.31s, 0 skips, live ClickHouse execution using canonical migration file**).
  - `go test -v -count=1 -cover ./internal/api/... ./internal/auth/... ./internal/ingest/... ./internal/repository/clickhouse/...` -> Exited 0 (**83 passing test/subtest assertions, 0 skips, 0 failures**).
  - **Statement Coverage on Touched Systems (Target >= 50%):**
    - `internal/ingest`: **66.1%**
    - `internal/api`: **60.9%**
    - `internal/auth`: **56.3%**
    - `internal/repository/clickhouse`: **51.7%**
  - `go build ./...` -> Exited 0.
  - `go build -o bin/server.exe ./cmd/server` -> Exited 0.
  - `npx eslint . --ext ts,tsx` -> Exited 0 (0 errors, 0 warnings).
  - `npm run build` -> Exited 0 (`dist/` built in 726ms).
- **Next Checkpoint's Exit Criteria (Phase 2 — Ingestion & Pipeline Robustness):**
  1. Kafka/Redpanda consumer integration verification (`internal/ingest/consumer.go`).
  2. Integration tests for trace ingestion, batch ClickHouse commits, and tenant isolation validation.
  3. Settings tab API key creation & revocation wired to PostgreSQL backend.

---

## Checkpoint: 2026-09-27 — Phase 2 Complete: Ingestion & Pipeline Robustness
- **Status:** ✅ **Phase 2 Complete — All Exit Criteria Satisfied**.
- **Project Completion:** ~82% (Full end-to-end telemetry ingestion pipeline, Kafka batch consumer, atomic offset commits, and connected Settings tab).
- **What Changed Since Last Checkpoint:**
  1. **Decoupled Ingestion Pipeline (Decision #19):**
     - Updated [`internal/api/spans.go`](file:///d:/Projects/DistributedTrace/internal/api/spans.go) to use `SpanPublisher` interface.
     - When `producer != nil`, `POST /api/v1/spans` publishes exclusively to Kafka, eliminating the $2\times$ duplicate write bug.
     - Direct ClickHouse insert preserved as a clean fallback when `producer == nil` (enabling zero-dependency local dev).
     - Added comprehensive unit tests in [`internal/api/spans_test.go`](file:///d:/Projects/DistributedTrace/internal/api/spans_test.go) verifying producer active, producer error (500), ClickHouse fallback, and schema validation.
  2. **Kafka/Redpanda Batch Consumer (`internal/ingest/consumer.go`):**
     - Created [`internal/ingest/consumer.go`](file:///d:/Projects/DistributedTrace/internal/ingest/consumer.go) providing `SpanConsumer` with configurable `BatchSize` (500 spans) and `FlushInterval` (1s).
     - Decoupled `MessageReader` and `SpanBatchWriter` interfaces for testability and clean architecture.
     - Implemented atomic offset commits strictly after successful ClickHouse `InsertSpansBatch` (preventing silent data loss on database connection drops).
     - Poison pill handling: commits malformed payloads without halting partition consumption.
     - Graceful shutdown: flushes remaining buffer to ClickHouse and commits offsets upon context cancellation.
     - Added unit tests in [`internal/ingest/consumer_test.go`](file:///d:/Projects/DistributedTrace/internal/ingest/consumer_test.go) verifying batch size flush, ticker flush, offset safety on ClickHouse failure, and malformed payload commit.
  3. **Live Redpanda Integration Test & Offset Safety Verification:**
     - Created [`internal/ingest/redpanda_integration_test.go`](file:///d:/Projects/DistributedTrace/internal/ingest/redpanda_integration_test.go) (`TestSpanConsumer_RealRedpanda_OffsetSafetyAndRestart`).
     - Produces real spans to a live Redpanda topic via `SpanProducer`.
     - Simulates mid-batch downstream database failure on Consumer 1, proving offsets are not committed.
     - Restarts Consumer 2 on the same consumer group, proving in-flight spans are re-delivered and persisted without loss.
     - Starts Consumer 3 on the same consumer group, proving zero spans are re-processed once offsets are committed.
     - Configured Redpanda service startup step in [`.github/workflows/ci.yml`](file:///d:/Projects/DistributedTrace/.github/workflows/ci.yml) to execute in remote CI alongside ClickHouse.
  4. **Exponential Backoff on Downstream Failures (Decision #21):**
     - Implemented exponential backoff with capped jitter (200ms up to 10s) in [`internal/ingest/consumer.go`](file:///d:/Projects/DistributedTrace/internal/ingest/consumer.go) on consecutive ClickHouse persistence failures, preventing CPU spinning and log spam during extended outages.
  5. **End-to-End Pipeline & Multi-Tenant Routing Test:**
     - Created [`internal/ingest/pipeline_test.go`](file:///d:/Projects/DistributedTrace/internal/ingest/pipeline_test.go) (`TestEndToEndIngestionPipeline_TenantIsolationAndBatchCommit`).
     - Simulates multi-tenant span batches from Org A and Org B, passes through Kafka message transport, exercises consumer batching and ClickHouse persistence, and verifies offset commitment and tenant isolation in persisted batch rows.
  6. **Live Settings Tab API Key Management (`src/components/dashboard/SettingsTab.tsx`):**
     - Replaced hardcoded mockup generator and placeholder rows in [`src/components/dashboard/SettingsTab.tsx`](file:///d:/Projects/DistributedTrace/src/components/dashboard/SettingsTab.tsx) with `@tanstack/react-query` hooks.
     - Connected to backend endpoints: `GET /api/v1/auth/api-keys`, `POST /api/v1/auth/api-keys`, and `DELETE /api/v1/auth/api-keys/{id}`.
     - Modal form allows naming keys and choosing between Production (`dt_live_...`) and Test (`dt_test_...`) scopes.
     - One-time secret reveal dialog with copy button and cURL invocation example.
     - Interactive revocation with confirmation dialog, loading spinners, and active/revoked badges.
  7. **Async Ingestion UX Polling & Pipeline Drain Feedback:**
     - Updated [`src/components/dashboard/TracesTab.tsx`](file:///d:/Projects/DistributedTrace/src/components/dashboard/TracesTab.tsx) with `pipelineFlushing` state and staggered query invalidation (immediate, +1.2s, +2.5s) to smoothly handle the Kafka buffer flush window, preventing empty flash when sending sample spans.
  8. **Server Lifecycle & Observability Integration:**
     - Wired `SpanProducer` and `SpanConsumer` into [`cmd/server/main.go`](file:///d:/Projects/DistributedTrace/cmd/server/main.go) under `cfg.KafkaBrokers`.
     - Integrated consumer buffer draining and producer close into graceful server shutdown.
     - Added `KafkaMessagesConsumed` counter and `KafkaConsumerBatchDuration` histogram to [`internal/observability/metrics.go`](file:///d:/Projects/DistributedTrace/internal/observability/metrics.go).
     - Initialized global `Log` to `zap.NewNop()` in [`internal/observability/logger.go`](file:///d:/Projects/DistributedTrace/internal/observability/logger.go) to safeguard against uninitialized logging in tests.
- **Verification Evidence (Commands Run & CI Runs):**
  - **Remote CI Run #7 (Head SHA `5c0e44e`):** **SUCCESS (All Green)**
    - Run URL: [https://github.com/RaviBharathi410/Distributed-Trace/actions/runs/36334541422](https://github.com/RaviBharathi410/Distributed-Trace/actions/runs/36334541422)
    - Frontend Job: `Frontend (Node 20 / Vite)` -> **Success** (ESLint + TypeScript typecheck + Vite build in 22s).
    - Backend Job: `Backend (Go 1.22)` -> **Success** (Go build + Standalone server + Redpanda cluster ready + Live unit, regression, and dual-engine ClickHouse + Redpanda integration tests with race detector and coverage in 1m 27s).
  - **Diagnostic & Fix Summary for Remote CI Failure (Run #6 -> Run #7):**
    1. **ClickHouse Strict Column Scan Coercion (`internal/repository/clickhouse/traces.go`):** In `SearchTraces`, `count()` and `sum(status_code = 2)` return `UInt64` in ClickHouse. `clickhouse-go/v2` rejected scanning `UInt64` directly into signed Go `*int` pointers (`&tr.SpanCount`, `&tr.ErrorCount`). Added intermediate `uint64` buffer variables and cast cleanly into `TraceRow`.
    2. **Kafka Dynamic Topic Auto-Creation (`internal/ingest/producer.go`):** `kafka.Writer` defaults `AllowAutoTopicCreation` to `false`. Added `AllowAutoTopicCreation: true` to enable dynamic test topic registration against Redpanda.
    3. **Consumer Latency (`internal/ingest/consumer.go`):** Set `MinBytes: 1` (was `10e3`) so low-volume span batches consume immediately without idling for `MaxWait`.
    4. **Redpanda Cluster Healthiness in CI (`.github/workflows/ci.yml`):** Replaced port-only `nc -z` probe with `docker compose exec -T redpanda rpk cluster health | grep -q 'HEALTHY'`, guaranteeing internal Raft leader election before test execution.
    5. **Test Retries & Timing (`internal/ingest/redpanda_integration_test.go`):** Added a 5-attempt retry loop on initial `producer.PublishSpans` to account for partition leader metadata discovery, and extended consumer restart wait deadlines.
  - **Local Verification:**
    - `go test -v -count=1 -cover ./internal/api/... ./internal/auth/... ./internal/ingest/... ./internal/repository/clickhouse/...` -> Exited 0 (**All tests passing, 0 failures**).
    - Statement coverage: `internal/ingest`: **69.0%**, `internal/api`: **62.1%**, `internal/auth`: **56.3%**, `internal/repository/clickhouse`: **51.7%**.
    - `go build ./...` -> Exited 0.
    - `go build -o bin/server.exe ./cmd/server` -> Exited 0.
    - `npx eslint . --ext ts,tsx` -> Exited 0 (0 errors, 0 warnings).
    - `npm run build` -> Exited 0 (`dist/` built cleanly in 3.88s).
- **Next Checkpoint's Exit Criteria (Phase 3 — Causal Analysis & Anomaly Detection Pipeline):**
  1. Statistical baseline calculation for service latency (mean, standard deviation, z-score detection).
  2. Causal graph traversal identifying root cause service when downstream latency anomalies occur.
  3. Anomaly persistence in ClickHouse `anomalies` table with multi-tenant partitioning.
  4. End-to-end integration test verifying anomaly detection from ingested span latency spikes.

---

## Checkpoint: 2026-09-30 — Phase 3 Complete: Causal Analysis & Anomaly Detection Pipeline
- **Status:** ✅ **Phase 3 Complete — All Exit Criteria Satisfied**.
- **Project Completion:** ~89% (Causal graph traversal, Welford statistical baselines, ClickHouse anomaly persistence, consumer hook wiring, and live integration tests).
- **What Changed Since Last Checkpoint:**
  1. **Statistical Baseline Calculation ([`internal/analysis/baseline.go`](file:///d:/Projects/DistributedTrace/internal/analysis/baseline.go)):**
     - Implemented Welford's single-pass algorithm for rolling mean $\mu$ and sample variance $\sigma^2 = M_2 / (N-1)$.
     - Added cold-start guard ($N \ge 5$ required before anomaly classification).
     - Standard deviation floor ($\sigma \ge 1.0\text{ms}$) to prevent division by zero and microsecond false positives on uniform workloads.
     - Minimum absolute duration threshold ($\ge 50\text{ms}$) and minimum deviation threshold ($d - \mu \ge 30\text{ms}$) to avoid alerting on trivial sub-millisecond jitter.
     - Historical baseline preloading via `PreloadFromClickHouse` querying quantiles and averages.
     - Comprehensive unit tests in [`internal/analysis/baseline_test.go`](file:///d:/Projects/DistributedTrace/internal/analysis/baseline_test.go).
  2. **Deterministic Causal Graph Traversal ([`internal/analysis/causal.go`](file:///d:/Projects/DistributedTrace/internal/analysis/causal.go)):**
     - Assembles in-memory trace trees from raw spans grouped by `(OrgID, TraceID)`.
     - Computes self-time for each span: $\text{SelfTime} = \text{Duration} - \sum \text{ChildDurations}$.
     - Critical-path bottleneck detection: evaluates downstream children consuming $\ge 40\%$ parent duration, walking down the critical path until the leaf bottleneck or highest self-time node is identified as `RootCauseService` with ordered `RootCausePath`.
     - Pure algorithmic execution ($O(N)$) with zero LLM calls on the hot path (Decision #26).
     - Verified with unit tests in [`internal/analysis/causal_test.go`](file:///d:/Projects/DistributedTrace/internal/analysis/causal_test.go).
  3. **Anomaly Detection Orchestrator ([`internal/analysis/detector.go`](file:///d:/Projects/DistributedTrace/internal/analysis/detector.go)):**
     - Orchestrates baseline evaluation, causal tree analysis, severity grading (`critical`, `high`, `medium`, `low`), and error-rate delta calculation.
     - Persists anomalies to ClickHouse `anomalies` table with strict multi-tenant partitioning.
     - Full unit test coverage in [`internal/analysis/detector_test.go`](file:///d:/Projects/DistributedTrace/internal/analysis/detector_test.go).
  4. **ClickHouse Anomaly Persistence & Scan Safety (Decision #24):**
     - Fixed `GetStats` in [`internal/repository/clickhouse/anomalies.go`](file:///d:/Projects/DistributedTrace/internal/repository/clickhouse/anomalies.go) to scan ClickHouse `sum(...)` into intermediate `uint64` variables before casting to `int`, resolving driver type mismatch errors.
     - Updated [`internal/repository/clickhouse/tenancy_test.go`](file:///d:/Projects/DistributedTrace/internal/repository/clickhouse/tenancy_test.go) mock helper `assignVal` to handle `uint64` targets.
  5. **Pipeline Integration & Consumer Hook ([`internal/ingest/consumer.go`](file:///d:/Projects/DistributedTrace/internal/ingest/consumer.go), [`cmd/server/main.go`](file:///d:/Projects/DistributedTrace/cmd/server/main.go)):**
     - Added `PostBatchHook` callback to `SpanConsumer` with a bounded worker pool (2 workers, queue capacity 32) and non-blocking load shedding (`AnalysisBatchesDropped`) to protect ingestion throughput and ClickHouse connection pools from saturation during traffic spikes.
     - Graceful drain on shutdown: `SpanConsumer.Run` drains queued analysis batches and waits for workers before closing.
     - Wired `AnomalyDetector.ProcessBatch` to `PostBatchHook` in `main.go`, seamlessly analyzing live ingested spans without blocking Kafka ingestion or ClickHouse batch commits.
  6. **Live Multi-Tier Integration Test ([`internal/analysis/integration_test.go`](file:///d:/Projects/DistributedTrace/internal/analysis/integration_test.go)):**
     - Executes against live ClickHouse instance loading DDL from migration files (`001_create_otel_spans.sql`, `002_create_anomalies.sql`).
     - Simulates a 3-tier distributed transaction (`api-gateway` -> `order-service` -> `catalog-db`), with a 1800ms bottleneck at `catalog-db`.
     - Asserts that an anomaly is created, `RootCauseService` is isolated to `catalog-db`, and strict tenant isolation holds (Org B cannot read Org A's anomalies).
  7. **CI Workflow Updated ([`.github/workflows/ci.yml`](file:///d:/Projects/DistributedTrace/.github/workflows/ci.yml)):**
     - Added `./internal/analysis/...` to test matrix with `-race` and `-cover`.
- **Verification Evidence (Commands Run & CI Runs):**
  - **Remote GitHub Actions CI Run #11:** [Run 36861862548](https://github.com/RaviBharathi410/Distributed-Trace/actions/runs/36861862548) -> **COMPLETED SUCCESS** (Head SHA `1f0b355`, Duration 1m 58s, 0 failures across both jobs).
    - `Backend (Go 1.22)` -> **Success** (Native ClickHouse container + Redpanda test broker + bounded worker pool tests + live integration tests).
    - `Frontend (Node 20 / Vite)` -> **Success** (ESLint + TypeScript typecheck + Vite build in 21s).
  - **Remote GitHub Actions CI Run #10:** [Run 36737145193](https://github.com/RaviBharathi410/Distributed-Trace/actions/runs/36737145193) -> **COMPLETED SUCCESS** (Head SHA `c071b8a`, Duration 1m 57s).
  - `go test -v -cover ./internal/analysis/...` -> Exited 0 (**100% tests passing**).
  - `go test -v -cover ./internal/ingest/...` -> Exited 0 (**70.2% statement coverage, 100% tests passing**).
  - `go test -v -cover ./internal/repository/clickhouse/...` -> Exited 0 (**100% tests passing**).
  - `go test ./...` -> Exited 0 (**All packages compile and pass cleanly**).
  - `npm run build` -> Exited 0 (`dist/` built in 3.92s).
- **Next Checkpoint's Exit Criteria (Phase 4 — Real-time Service Graph & Topology Visualization):**
  1. Service graph API aggregation endpoint (`GET /api/v1/services/graph`) computing live health status (`healthy`, `degraded`, `critical`) directly from active ClickHouse anomalies and latency baselines (unified truth model, no parallel computation path).
  2. Interactive Service Map frontend rendering node health status, RPS, error rates, and directed edge latencies.
  3. Real-time topology updates reflecting live ingested span metadata.


