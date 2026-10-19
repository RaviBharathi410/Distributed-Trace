# DistributedTrace Empirical Ingestion & Storage Benchmark Report

**Date:** 2026-10-19T11:29:03Z  
**Hardware Environment:** `Shared 2-4 vCPU box (GitHub Actions Runner / linux amd64) - INDICATIVE ONLY, NOT A SIZING GUIDE`  
**Dataset:** 1000000 spans sent (1000000 persisted) in 8192-span batches  
**Sustained Throughput:** `5740 spans/second` (duration: `174.22s`)  

## 1. ClickHouse Partition Consolidation (system.parts)

| Partition | Active Parts (Post-OPTIMIZE) | Rows | Bytes on Disk | Single Part Verified |
|---|---|---|---|---|
| `20261019` | 1 | 1000000 | **235052552 B** | ✅ Yes (1 part) |

## 2. Multi-Table Storage Footprint & Classification

| Table | Role / Classification | Active Parts | Rows | Compressed Data | Bytes on Disk |
|---|---|---|---|---|---|
| `otel_spans` | Primary Ingest (Raw Spans + Indexes) | 1 | 1000000 | 235045296 B | **235052552 B** |
| `service_metrics_hourly` | Auxiliary Aggregate Rollup (SummingMergeTree) | 0 | 0 | 0 B | **0 B** |
| `anomalies` | Auxiliary Incident Alerts (MergeTree) | 6 | 18309 | 1402719 B | **1409770 B** |

> [!NOTE]
> Telemetry ingest storage is accounted for primarily by `otel_spans` (columns + skipping indices + marks). The `anomalies` table is populated asynchronously by the Go server's background anomaly detector evaluating spans against statistical thresholds; with an 8% injected error rate and latency jitter, the detector recorded ~1.8% of spans as anomalies (~1.4 B/span disk footprint). `service_metrics_hourly` is an analytical rollup table (0 B during raw ingest).

## 3. Data Skipping Indices (Bloom Filter)

| Table | Index Name | Type | Expression | Granularity |
|---|---|---|---|---|
| `otel_spans` | `idx_service` | `bloom_filter` | `service_name` | 4 |

## 4. Top Column Compressed Footprint (otel_spans)

| Column | Compressed Bytes | Uncompressed Bytes | Compression Ratio |
|---|---|---|---|
| `tags` | 157258612 B | 580567589 B | 3.7x |
| `trace_id` | 33110515 B | 33000000 B | 1.0x |
| `span_id` | 17061457 B | 17000000 B | 1.0x |
| `parent_span_id` | 17061418 B | 17000000 B | 1.0x |
| `start_time` | 4042600 B | 8000000 B | 2.0x |
| `duration_ms` | 2936779 B | 4000000 B | 1.4x |
| `operation_name` | 2630201 B | 21229848 B | 8.1x |
| `error_message` | 437257 B | 6021633 B | 13.8x |
| `status_code` | 278777 B | 1000000 B | 3.6x |
| `org_id` | 153998 B | 37000000 B | 240.3x |

## 5. Empirical SaaS Cost-Per-Million-Spans Economics

- **All-Inclusive Disk Footprint per Span:** `236.46 bytes` (columns + skipping indices + marks)
- **Physical Disk Footprint per 1M Spans:** `0.220 GB`
- **Monthly Storage Cost (AWS EBS gp3):** `$0.0176 / million spans` ($0.08/GB-mo)
- **Monthly Storage Cost (AWS S3 Tiered):** `$0.0051 / million spans` ($0.023/GB-mo)
- **Cross-AZ Network Wire Ingress:** `$0.00142 / million spans` (0.142 GB wire payload)
- **Direct Measured Unit Cost:** `~$0.01904 / million spans` (Direct Storage + Wire)

> [!IMPORTANT]
> **Hardware Disclaimer & Pricing Guardrail:** Results are indicative of GitHub runner hardware (shared 2-4 vCPU box). Final SaaS pricing tiers are strictly held pending dedicated cloud VM production sizing.

## 6. Sustained Host & Container Resource Utilization (Time-Series Sampled)

| Process / Container | Peak CPU % | Avg CPU % | Peak RAM (MB) | Avg RAM (MB) | Samples |
|---|---|---|---|---|---|
| `8b17d2f8885c44cc87f2fae91bab618b_postgres16alpine_40e043` | **4.35%** | 0.96% | **35.9 MB** | 35.5 MB | 39 |
| `bafbc697e4a642e08a9b83d67cff0c01_clickhouseclickhouseserver243alpine_3a7db7` | **159.72%** | 79.24% | **2358.3 MB** | 1489.5 MB | 39 |
| `dt-redpanda` | **12.69%** | 4.90% | **910.1 MB** | 750.2 MB | 39 |
| `server-process` (Go Ingest/Consumer) | **49.70%** | 45.80% | **128.6 MB** | 105.6 MB | 39 |

> [!NOTE]
> Continuous background sampling captured every 2s during sustained ingestion load.
> **Process Sizing Disclaimer:** The server binary (`bin/server`) and the load generator (`bin/loadtest`) run directly on the host runner and share the runner's 2–4 vCPUs. The load generator's CPU footprint is co-located on the same box, so host resource readings are indicative of shared test-runner overhead, not dedicated production VM sizing.

