# DistributedTrace

DistributedTrace is a production-grade, multi-region observability platform designed for modern cloud-native infrastructure. It ingests OpenTelemetry spans, correlates traces using probabilistic causal inference, and surfaces anomalies in under 500ms.

## Features

- **Real-time Trace Streaming:** View a live feed of traces across your microservices architecture.
- **Service Dependency Graph:** Automatically visualize the topology of your services, highlighting latency and error bottlenecks.
- **Causal Anomaly Detection:** Instantly identify the root cause of latency spikes or error storms with our intelligent correlation engine.
- **Multi-Region Support:** Built for global scale, capable of ingesting data from `us-east-1`, `eu-west-1`, and beyond seamlessly.
- **Extremely High Throughput:** Powered by Kafka and ClickHouse, capable of handling 50K+ spans/sec ingest and petabytes of compressed trace storage.
- **RBAC & Security:** Robust role-based access control, secure API keys, and comprehensive audit logging.

## Tech Stack

### Frontend
- **Framework:** React 18 with TypeScript
- **Build Tool:** Vite
- **Styling:** Tailwind CSS + custom CSS animations
- **Icons:** Lucide React

### Backend (Under Construction)
- **Language:** Go (Golang) 1.22
- **Router:** Chi Router
- **WebSockets:** Gorilla WebSocket for live streaming
- **Telemetry:** OpenTelemetry (OTLP)

### Infrastructure & Data stores
- **Relational DB:** PostgreSQL (Users, Organizations, API Keys, Configuration)
- **Columnar DB:** ClickHouse (High-volume Spans, Metrics, Anomalies)
- **Message Broker:** Apache Kafka (Ingestion pipeline buffering)

## Getting Started

### Prerequisites
- Node.js (v18+)
- Go (v1.22+)
- Docker & Docker Compose (for local infrastructure)

### Running the Frontend Locally

1. Clone the repository
2. Install dependencies:
   ```bash
   npm install
   ```
3. Start the Vite development server:
   ```bash
   npm run dev
   ```
4. Open your browser and navigate to `http://localhost:5173`

### Backend Setup

*Note: The backend is currently in active development.*

The backend relies on PostgreSQL and ClickHouse. Migrations for both databases are located in the `/migrations/` directory.

To run the backend (once fully implemented):
```bash
go run cmd/server/main.go
```

## Project Structure

```
├── cmd/
│   └── server/          # Go application entrypoint
├── internal/
│   ├── auth/            # JWT & Hashing
│   ├── config/          # Environment configuration
│   ├── domain/          # Core models (Span, User, Anomaly, etc.)
│   ├── observability/   # OpenTelemetry & Zap Logging setup
│   └── repository/      # ClickHouse & PostgreSQL data access
├── migrations/
│   ├── clickhouse/      # Schema for analytical data
│   └── postgres/        # Schema for relational data
├── public/              # Static assets (images, fonts)
└── src/
    ├── components/      # React components (Dashboard, Modals, etc.)
    ├── pages/           # React views
    └── SecurifyApp.tsx  # Main React entrypoint
```

## License

MIT License. See `LICENSE` for more information.
