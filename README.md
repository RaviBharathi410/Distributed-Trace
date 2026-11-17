# DistributedTrace

DistributedTrace is a production-grade, multi-tenant observability platform designed for modern cloud-native infrastructure. It ingests OpenTelemetry spans, correlates traces using deterministic Welford statistical baselines and trace critical-path graph analysis, and provides transparent cost-attributed incident explanations.

## Features

- **High-Throughput Trace Ingestion:** OpenTelemetry (OTLP) HTTP Protobuf and Gzip ingest buffered through Kafka/Redpanda and persisted into ClickHouse MergeTree storage with `CODEC(ZSTD(3))` and binary `FixedString(16)` trace IDs.
- **Service Dependency Graph:** Dynamic service topology visualization highlighting latency and error bottlenecks, animated particle traffic flow, and live sparkline metrics.
- **Deterministic Causal Anomaly Detection:** Real-time root-cause bottleneck traversal and online Welford variance tracking with cold-start guards and tenant isolation.
- **On-Demand AI Root Cause Explanation:** User-triggered incident explanation powered by Gemini 3.5 Flash-Lite with two-tiered output validation (no remediation commands or soft advisories) and 24-hour fingerprint caching.
- **Tenant Cost Ledger & Guardrails:** Pre-flight token cost checks (≤ $0.010/call ceiling) and hourly storm budget caps ($1.00/hr) with transparent per-tenant spend attribution and vertical daily spend trend analytics.
- **Multi-Tenant RBAC & Security:** Organization isolation, Argon2id password hashing, Ed25519 asymmetric JWT authentication, and secure API key management.
- **Modern Dark-Mode Design System:** Source-matched pitch-black interface (`#0a0b14`) built with self-hosted Inter & JetBrains Mono typography, custom SVG data visualizations, and seamless bidirectional marketing landing page navigation.

## Tech Stack

### Frontend
- **Framework:** React 18 with TypeScript
- **Build Tool:** Vite
- **Styling:** Tailwind CSS + custom CSS animations
- **Icons:** Lucide React

### Backend
- **Language:** Go (Golang) 1.22
- **Router:** Chi Router
- **Telemetry:** OpenTelemetry (OTLP JSON & Protobuf)
- **Real-Time Updates:** Adaptive interval polling (10s auto-refresh, window-blur suppression) eliminating stateful WebSocket server overhead (Decision #29)

### Infrastructure & Data Stores
- **Relational DB:** PostgreSQL (Users, Organizations, API Keys, Tenant Configuration)
- **Columnar DB:** ClickHouse (High-volume Spans, Metrics, Anomalies)
- **Message Broker:** Apache Kafka / Redpanda (Ingestion pipeline stream buffer)
- **Caching & Baselines:** Redis / In-memory concurrent safe caches

## Getting Started

### Prerequisites
- Node.js (v18+)
- Go (v1.22+)
- Docker & Docker Compose

### 1. Start Infrastructure Dependencies
```bash
docker compose up -d
```
This starts PostgreSQL, ClickHouse, Redpanda (Kafka), and Redis.

### 2. Run the Backend API Server
```bash
go run cmd/server/main.go
```
The server starts listening on `:8080`.

### 3. Run the Frontend Development Server
```bash
npm install
npm run dev
```
Open your browser and navigate to `http://localhost:5173`.

## Project Structure

```
├── cmd/
│   ├── e2e-smoke/            # End-to-end integration smoke test with real OTLP ingest
│   ├── loadtest/             # 20M-span vectorized ClickHouse scale benchmark
│   ├── migrate/              # Database migration runner (PostgreSQL & ClickHouse)
│   └── server/               # Core DistributedTrace API server
├── docs/
│   └── screenshots/          # 1440x900 verified screenshots across all screens
├── internal/
│   ├── analysis/             # Causal graph analysis, Welford baselines & LLM explainer
│   ├── api/                  # HTTP REST endpoints & OTLP receivers
│   ├── auth/                 # Argon2id, Ed25519 JWT & API key verification
│   ├── config/               # Environment configuration & cost constants
│   ├── domain/               # Core models (Span, Anomaly, User, Organization)
│   ├── ingest/               # OTLP parser (Protobuf/Gzip), Kafka producer & consumer
│   ├── observability/        # Zap logging & Prometheus metrics
│   └── repository/           # ClickHouse (ZSTD(3), binary IDs) & PostgreSQL data access
├── migrations/
│   ├── clickhouse/           # MergeTree schemas with ZSTD(3) and FixedString binary IDs
│   └── postgres/             # Schema for relational user & tenant data
└── src/
    ├── components/
    │   ├── common/           # Shared design-system primitives (StatCard, Waterfall, Donut, etc.)
    │   ├── dashboard/        # Operations tabs (Traces, ServiceMap, Anomalies, Costs, Settings)
    │   └── shell/            # AppShell, BrandLogo & GlobalSearch (Ctrl+K)
    ├── pages/                # LandingPage, OverviewPage, TracesPage, ServiceMapPage, AnomaliesPage, CostsPage, SettingsPage
    ├── styles/               # Strict dark design tokens (tokens.css) & self-hosted fonts
    └── App.tsx               # Client router with bidirectional landing and dashboard linkage
```

## License

MIT License. See `LICENSE` for more information.
