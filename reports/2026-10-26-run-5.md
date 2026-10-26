# DistributedTrace Empirical Ingestion & Storage Benchmark Report

**Date:** 2026-10-26T14:31:13Z  
**Hardware Environment:** `Shared 2-4 vCPU box (GitHub Actions Runner / linux amd64) - INDICATIVE ONLY, NOT A SIZING GUIDE`  
**Dataset:** 1000000 spans sent (1000000 persisted) in 8192-span batches  
**Sustained Throughput:** `6143 spans/second` (duration: `162.79s`)  

## 1. ClickHouse Partition Consolidation (system.parts)

| Partition | Active Parts (Post-OPTIMIZE) | Rows | Bytes on Disk | Single Part Verified |
|---|---|---|---|---|
| `20261026` | 1 | 1000000 | **235132053 B** | ✅ Yes (1 part) |

## 2. Multi-Table Storage Footprint & Classification

| Table | Role / Classification | Active Parts | Rows | Compressed Data | Bytes on Disk |
|---|---|---|---|---|---|
| `otel_spans` | Primary Ingest (Raw Spans + Indexes) | 1 | 1000000 | 235124807 B | **235132053 B** |
| `service_metrics_hourly` | Auxiliary Aggregate Rollup (SummingMergeTree) | 0 | 0 | 0 B | **0 B** |
| `anomalies` | Auxiliary Incident Alerts (MergeTree) | 3 | 18613 | 1424454 B | **1428082 B** |

> [!NOTE]
> Telemetry ingest storage is accounted for primarily by `otel_spans` (columns + skipping indices + marks). The `anomalies` table is populated asynchronously by the Go server's background anomaly detector evaluating spans against statistical thresholds; with an 8% injected error rate and latency jitter, the detector recorded ~1.8% of spans as anomalies (~1.4 B/span disk footprint). `service_metrics_hourly` is an analytical rollup table (0 B during raw ingest).

## 3. Data Skipping Indices (Bloom Filter)

| Table | Index Name | Type | Expression | Granularity |
|---|---|---|---|---|
| `otel_spans` | `idx_service` | `bloom_filter` | `service_name` | 4 |

## 4. Top Column Compressed Footprint (otel_spans)

| Column | Compressed Bytes | Uncompressed Bytes | Compression Ratio |
|---|---|---|---|
| `tags` | 157332935 B | 580914173 B | 3.7x |
| `trace_id` | 33111874 B | 33000000 B | 1.0x |
| `span_id` | 17062611 B | 17000000 B | 1.0x |
| `parent_span_id` | 17061789 B | 17000000 B | 1.0x |
| `start_time` | 4042636 B | 8000000 B | 2.0x |
| `duration_ms` | 2935833 B | 4000000 B | 1.4x |
| `operation_name` | 2630809 B | 21230374 B | 8.1x |
| `error_message` | 439608 B | 6062292 B | 13.8x |
| `status_code` | 279034 B | 1000000 B | 3.6x |
| `org_id` | 153998 B | 37000000 B | 240.3x |

## 5. Empirical SaaS Cost-Per-Million-Spans Economics

- **All-Inclusive Disk Footprint per Span:** `236.56 bytes` (columns + skipping indices + marks)
- **Physical Disk Footprint per 1M Spans:** `0.220 GB`
- **Monthly Storage Cost (AWS EBS gp3):** `$0.0176 / million spans` ($0.08/GB-mo)
- **Monthly Storage Cost (AWS S3 Tiered):** `$0.0051 / million spans` ($0.023/GB-mo)
- **Cross-AZ Network Wire Ingress:** `$0.00142 / million spans` (0.142 GB wire payload)
- **Direct Measured Unit Cost:** `~$0.01905 / million spans` (Direct Storage + Wire)

> [!IMPORTANT]
> **Hardware Disclaimer & Pricing Guardrail:** Results are indicative of GitHub runner hardware (shared 2-4 vCPU box). Final SaaS pricing tiers are strictly held pending dedicated cloud VM production sizing.

## 6. Baseline Query Latency & Deterministic I/O (20000000 Rows Scale)

### 6.1 Query Performance & Deterministic I/O Summary (20 Iterations)

| Query Operation | Scale | First Run (Cache Dropped) | P50 Latency | P95 Latency | Cold Read Rows | Avg Read Rows | Cold Read Bytes | Avg Read Bytes | Status / Analysis |
|---|---|---|---|---|---|---|---|---|---|
| `GetTraceWithSpans (Point Lookup)` | 20M rows | **213 ms** | **209.0 ms** | **237.0 ms** | 20000000 | 20000000 | 1720652696 B | 1722082692 B | ⚠️ Scans full partition (no bloom filter index on trace_id yet) |
| `SearchTraces (Filter & Sort)` | 20M rows | **9 ms** | **7.0 ms** | **9.0 ms** | 41599 | 41689 | 9475205 B | 9493590 B | ✅ Primary key pruned via (org_id, service_name, start_time) |
| `GetServiceGraph (Self-Join)` | 20M rows | **28 ms** | **36.0 ms** | **56.0 ms** | 892928 | 1931156 | 67203808 B | 144463894 B | ⚠️ Self-join scan across time window |

> [!NOTE]
> **Deterministic Metrics:** `read_rows` and `read_bytes` are recorded directly from ClickHouse `system.query_log` after `SYSTEM FLUSH LOGS`. Wall-clock latency on shared 2–4 vCPU GitHub Actions runners exhibits jitter, whereas read rows and read bytes are deterministic proof of part pruning and index skipping.
> **Cold Cache Protocol:** `SYSTEM DROP MARK CACHE` and `SYSTEM DROP UNCOMPRESSED CACHE` were executed immediately prior to iteration 1 ("first run, cache dropped"). On every subsequent iteration, target `trace_id`, `service_name`, and timestamp windows were dynamically varied across the 20M dataset to prevent reading cached blocks.

### 6.2 Query Execution Plans (EXPLAIN)

#### `GetTraceWithSpans (Point Lookup)` Execution Plan

```text
Expression ((Project names + Projection))
  Expression
    ReadFromMergeTree (default.otel_spans)
    Indexes:
      MinMax
        Condition: true
        Parts: 2/2
        Granules: 2443/2443
      Partition
        Condition: true
        Parts: 2/2
        Granules: 2443/2443
      PrimaryKey
        Keys: 
          org_id
        Condition: (org_id in ['55634cd8-1562-4f74-ba7e-ab95c6b94bbf', '55634cd8-1562-4f74-ba7e-ab95c6b94bbf'])
        Parts: 2/2
        Granules: 2443/2443
```

#### `SearchTraces (Filter & Sort)` Execution Plan

```text
EXPLAIN failed: code: 184, message: Aggregate function min(start_time) AS start_time is found in WHERE in query
```

#### `GetServiceGraph (Self-Join)` Execution Plan

```text
Expression ((Project names + Projection))
  Aggregating
    Expression (Before GROUP BY)
      Filter ((WHERE + DROP unused columns after JOIN))
        Join (JOIN FillRightFirst)
          Expression
            ReadFromMergeTree (default.otel_spans)
            Indexes:
              MinMax
                Keys: 
                  start_time
                Condition: and((start_time in (-Inf, '1792369800']), (start_time in ['1792368000', +Inf)))
                Parts: 1/2
                Granules: 2320/2443
              Partition
                Keys: 
                  toYYYYMMDD(start_time)
                Condition: and((toYYYYMMDD(start_time) in (-Inf, 20261019]), (toYYYYMMDD(start_time) in [20261019, +Inf)))
                Parts: 1/1
                Granules: 2320/2320
              PrimaryKey
                Keys: 
                  org_id
                  start_time
                Condition: and((start_time in (-Inf, '1792369800']), and((start_time in ['1792368000', +Inf)), (org_id in ['55634cd8-1562-4f74-ba7e-ab95c6b94bbf', '55634cd8-1562-4f74-ba7e-ab95c6b94bbf'])))
                Parts: 1/1
                Granules: 109/2320
          Expression ((JOIN actions + Change column names to column identifiers))
            ReadFromMergeTree (default.otel_spans)
            Indexes:
              MinMax
                Condition: true
                Parts: 2/2
                Granules: 2443/2443
              Partition
                Condition: true
                Parts: 2/2
                Granules: 2443/2443
              PrimaryKey
                Condition: true
                Parts: 2/2
                Granules: 2443/2443
```


## 7. Sustained Host & Container Resource Utilization (Time-Series Sampled)

| Process / Container | Peak CPU % | Avg CPU % | Peak RAM (MB) | Avg RAM (MB) | Samples |
|---|---|---|---|---|---|
| `30937d218a6c457196753ed39f819b50_clickhouseclickhouseserver243alpine_5c532a` | **372.24%** | 108.95% | **6501.4 MB** | 2896.2 MB | 76 |
| `3ba5e361dab74d90955c566d51f8a537_postgres16alpine_a3fdb7` | **3.53%** | 0.63% | **30.3 MB** | 29.7 MB | 76 |
| `dt-redpanda` | **8.88%** | 2.21% | **910.9 MB** | 831.5 MB | 76 |
| `server-process` (Go Ingest/Consumer) | **37.60%** | 29.57% | **120.8 MB** | 87.4 MB | 76 |

> [!NOTE]
> Continuous background sampling captured every 2s during sustained ingestion load.
> **Process Sizing Disclaimer:** The server binary (`bin/server`) and the load generator (`bin/loadtest`) run directly on the host runner and share the runner's 2–4 vCPUs. The load generator's CPU footprint is co-located on the same box, so host resource readings are indicative of shared test-runner overhead, not dedicated production VM sizing.

