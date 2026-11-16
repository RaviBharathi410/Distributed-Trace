# DistributedTrace Empirical Ingestion & Storage Benchmark Report

**Date:** 2026-11-16T10:37:37Z  
**Hardware Environment:** `Shared 2-4 vCPU box (GitHub Actions Runner / linux amd64) - INDICATIVE ONLY, NOT A SIZING GUIDE`  
**Dataset:** 1000000 spans sent (1000000 persisted) in 8192-span batches  
**Sustained Throughput:** `6247 spans/second` (duration: `160.08s`)  

## 1. ClickHouse Partition Consolidation (system.parts)

| Partition | Active Parts (Post-OPTIMIZE) | Rows | Bytes on Disk | Single Part Verified |
|---|---|---|---|---|
| `20261116` | 1 | 1000000 | **137873731 B** | ✅ Yes (1 part) |

## 2. Multi-Table Storage Footprint & Classification

| Table | Role / Classification | Active Parts | Rows | Compressed Data | Bytes on Disk |
|---|---|---|---|---|---|
| `otel_spans` | Primary Ingest (Raw Spans + Indexes) | 1 | 1000000 | 136861762 B | **137873731 B** |
| `service_metrics_hourly` | Auxiliary Aggregate Rollup (SummingMergeTree) | 0 | 0 | 0 B | **0 B** |
| `anomalies` | Auxiliary Incident Alerts (MergeTree) | 6 | 18454 | 1414308 B | **1421361 B** |

> [!NOTE]
> Telemetry ingest storage is accounted for primarily by `otel_spans` (columns + skipping indices + marks). The `anomalies` table is populated asynchronously by the Go server's background anomaly detector evaluating spans against statistical thresholds; with an 8% injected error rate and latency jitter, the detector recorded ~1.8% of spans as anomalies (~1.4 B/span disk footprint). `service_metrics_hourly` is an analytical rollup table (0 B during raw ingest).

## 3. Data Skipping Indices (Bloom Filter)

| Table | Index Name | Type | Expression | Granularity |
|---|---|---|---|---|
| `otel_spans` | `idx_service` | `bloom_filter` | `service_name` | 4 |
| `otel_spans` | `idx_trace` | `bloom_filter` | `trace_id` | 1 |

## 4. Top Column Compressed Footprint (otel_spans)

| Column | Compressed Bytes | Uncompressed Bytes | Compression Ratio |
|---|---|---|---|
| `tags` | 94172222 B | 580492972 B | 6.2x |
| `trace_id` | 16065943 B | 16000000 B | 1.0x |
| `span_id` | 8034571 B | 8000000 B | 1.0x |
| `parent_span_id` | 8034571 B | 8000000 B | 1.0x |
| `start_time` | 4042558 B | 8000000 B | 2.0x |
| `duration_ms` | 2936290 B | 4000000 B | 1.4x |
| `operation_name` | 2632615 B | 21234619 B | 8.1x |
| `error_message` | 437031 B | 6024712 B | 13.8x |
| `status_code` | 278292 B | 1000000 B | 3.6x |
| `org_id` | 153998 B | 37000000 B | 240.3x |

### 4.1 Tags Column Codec Empirical Comparison (1M Spans: LZ4 vs ZSTD(1) vs ZSTD(3))

| Codec | Compressed Bytes | Uncompressed Bytes | Compression Ratio | Tags B/Span | Compression Time (1M Spans) | Storage Savings vs LZ4 |
|---|---|---|---|---|---|---|
| `LZ4 (Default Baseline)` | 157227432 B | 580492972 B | 3.69x | 157.23 B | 3232 ms | Baseline (0.0%) |
| `ZSTD(1)` | 95222548 B | 580492972 B | 6.10x | 95.22 B | 4016 ms | ✅ **-39.4%** |
| `ZSTD(3)` | 94172222 B | 580492972 B | 6.16x | 94.17 B | 4848 ms | ✅ **-40.1%** |

> [!NOTE]
> Empirical codec comparison executed directly on the 1,000,000 live ingested span tags dataset in ClickHouse, isolating data density vs compression CPU throughput.


## 5. Empirical SaaS Cost-Per-Million-Spans Economics

- **Physical Disk Footprint per Span:** `139.30 bytes` (columns + skipping indices + marks)
- **Physical Disk Footprint per 1M Spans:** `0.130 GB`
- **Monthly Storage Cost (AWS EBS gp3):** `$0.0104 / million spans / month` ($0.08/GB-mo)
- **Monthly Storage Cost (AWS S3 Tiered):** `$0.0030 / million spans / month` ($0.023/GB-mo)
- **One-Time Network Wire Ingress:** `$0.00142 / million spans` (0.142 GB wire payload at $0.01/GB)
- **Initial Month Combined Unit Cost (EBS + Ingress):** `~$0.01180 / million spans`
- **Initial Month Combined Unit Cost (S3 Tiered + Ingress):** `~$0.00441 / million spans`

> [!NOTE]
> Recurring monthly physical storage fees and one-time cross-AZ network wire ingress costs are distinct economic line items and are accounted for separately above.

> [!IMPORTANT]
> **Hardware Disclaimer & Pricing Guardrail:** Results are indicative of GitHub runner hardware (shared 2-4 vCPU box). Final SaaS pricing tiers are strictly held pending dedicated cloud VM production sizing.

## 6. Baseline Query Latency & Deterministic I/O (20000000 Rows Scale)

### 6.1 Query Performance & Deterministic I/O Summary (20 Iterations)

| Query Operation | Scale | First Run (Cache Dropped) | P50 Latency | P95 Latency | Cold Read Rows | Avg Read Rows | Cold Read Bytes | Avg Read Bytes | Result Rows | Status / Analysis |
|---|---|---|---|---|---|---|---|---|---|---|
| `GetTraceWithSpans (Point Lookup)` | 20M rows | **7 ms** | **7.0 ms** | **10.0 ms** | 41191 | 41281 | 9372915 B | 9389367 B | 0 (avg 0.0) | ✅ Pruned via bloom_filter index idx_trace on trace_id FixedString(16) |
| `SearchTraces (Single Service, Primary Key Pruned)` | 20M rows | **3 ms** | **6.0 ms** | **19.0 ms** | 8192 | 14228 | 670936 B | 1291334 B | 0 (avg 23.7) | ✅ Primary key pruned via (org_id, service_name, start_time) |
| `SearchTraces (All Services, Last 15 Min)` | 20M rows | **156 ms** | **5.0 ms** | **118.0 ms** | 1007616 | 169543 | 124918769 B | 14327469 B | 50 (avg 7.9) | ⚠️ Cross-service scan over 15 min window (no service filter) |
| `GetServiceGraph (Self-Join)` | 20M rows | **23 ms** | **5.0 ms** | **31.0 ms** | 1007616 | 169543 | 81968811 B | 11131504 B | 0 (avg 0.0) | ⚠️ Self-join scan across 15 min fixed window |

> [!NOTE]
> **Deterministic Metrics:** `read_rows` and `read_bytes` are recorded directly from ClickHouse `system.query_log` after `SYSTEM FLUSH LOGS`. Wall-clock latency on shared 2–4 vCPU GitHub Actions runners exhibits jitter, whereas read rows and read bytes are deterministic proof of part pruning and index skipping.
> **Cold Cache Protocol & Warm OS Cache:** `SYSTEM DROP MARK CACHE` and `SYSTEM DROP UNCOMPRESSED CACHE` were executed immediately prior to iteration 1 ("first run, cache dropped"). Iterations 2–20 operate with warm OS page cache and dynamically rotating parameters across the 20M dataset.

### 6.2 Query Execution Plans (EXPLAIN)

#### `GetTraceWithSpans (Point Lookup)` Execution Plan

```text
Expression ((Project names + Projection))
  Expression
    ReadFromMergeTree (default.otel_spans)
    Indexes:
      MinMax
        Condition: true
        Parts: 3/3
        Granules: 2442/2442
      Partition
        Condition: true
        Parts: 3/3
        Granules: 2442/2442
      PrimaryKey
        Keys: 
          org_id
        Condition: (org_id in ['8d346d61-065f-4af0-a130-3655beff9a24', '8d346d61-065f-4af0-a130-3655beff9a24'])
        Parts: 3/3
        Granules: 2442/2442
      Skip
        Name: idx_trace
        Description: bloom_filter GRANULARITY 1
        Parts: 3/3
        Granules: 74/2442
```

#### `SearchTraces (Single Service, Primary Key Pruned)` Execution Plan

```text
Expression (Project names)
  Limit (preliminary LIMIT (without OFFSET))
    Sorting (Sorting for ORDER BY)
      Expression ((Before ORDER BY + Projection))
        Aggregating
          Expression (Before GROUP BY)
            Expression
              ReadFromMergeTree (default.otel_spans)
              Indexes:
                MinMax
                  Keys: 
                    start_time
                  Condition: and((start_time in (-Inf, '1794791400']), (start_time in ['1794790800', +Inf)))
                  Parts: 1/3
                  Granules: 1011/2442
                Partition
                  Keys: 
                    toYYYYMMDD(start_time)
                  Condition: and((toYYYYMMDD(start_time) in (-Inf, 20261116]), (toYYYYMMDD(start_time) in [20261116, +Inf)))
                  Parts: 1/1
                  Granules: 1011/1011
                PrimaryKey
                  Keys: 
                    org_id
                    service_name
                    start_time
                  Condition: and((start_time in (-Inf, '1794791400']), and((start_time in ['1794790800', +Inf)), and((service_name in ['service-0', 'service-0']), (org_id in ['8d346d61-065f-4af0-a130-3655beff9a24', '8d346d61-065f-4af0-a130-3655beff9a24']))))
                  Parts: 1/1
                  Granules: 1/1011
                Skip
                  Name: idx_service
                  Description: bloom_filter GRANULARITY 4
                  Parts: 1/1
                  Granules: 1/1
```

#### `SearchTraces (All Services, Last 15 Min)` Execution Plan

```text
Expression (Project names)
  Limit (preliminary LIMIT (without OFFSET))
    Sorting (Sorting for ORDER BY)
      Expression ((Before ORDER BY + Projection))
        Aggregating
          Expression (Before GROUP BY)
            Expression
              ReadFromMergeTree (default.otel_spans)
              Indexes:
                MinMax
                  Keys: 
                    start_time
                  Condition: and((start_time in (-Inf, '1794791700']), (start_time in ['1794790800', +Inf)))
                  Parts: 1/3
                  Granules: 1011/2442
                Partition
                  Keys: 
                    toYYYYMMDD(start_time)
                  Condition: and((toYYYYMMDD(start_time) in (-Inf, 20261116]), (toYYYYMMDD(start_time) in [20261116, +Inf)))
                  Parts: 1/1
                  Granules: 1011/1011
                PrimaryKey
                  Keys: 
                    org_id
                    start_time
                  Condition: and((start_time in (-Inf, '1794791700']), and((start_time in ['1794790800', +Inf)), (org_id in ['8d346d61-065f-4af0-a130-3655beff9a24', '8d346d61-065f-4af0-a130-3655beff9a24'])))
                  Parts: 1/1
                  Granules: 123/1011
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
                Condition: and((start_time in (-Inf, '1794791700']), (start_time in ['1794790800', +Inf)))
                Parts: 1/3
                Granules: 1011/2442
              Partition
                Keys: 
                  toYYYYMMDD(start_time)
                Condition: and((toYYYYMMDD(start_time) in (-Inf, 20261116]), (toYYYYMMDD(start_time) in [20261116, +Inf)))
                Parts: 1/1
                Granules: 1011/1011
              PrimaryKey
                Keys: 
                  org_id
                  start_time
                Condition: and((start_time in (-Inf, '1794791700']), and((start_time in ['1794790800', +Inf)), (org_id in ['8d346d61-065f-4af0-a130-3655beff9a24', '8d346d61-065f-4af0-a130-3655beff9a24'])))
                Parts: 1/1
                Granules: 123/1011
          Expression ((JOIN actions + Change column names to column identifiers))
            ReadFromMergeTree (default.otel_spans)
            Indexes:
              MinMax
                Condition: true
                Parts: 3/3
                Granules: 2442/2442
              Partition
                Condition: true
                Parts: 3/3
                Granules: 2442/2442
              PrimaryKey
                Condition: true
                Parts: 3/3
                Granules: 2442/2442
```


## 7. Sustained Host & Container Resource Utilization (Time-Series Sampled)

| Process / Container | Peak CPU % | Avg CPU % | Peak RAM (MB) | Avg RAM (MB) | Samples |
|---|---|---|---|---|---|
| `a3c96a2b85e6417bb34a7eb9417bbd31_clickhouseclickhouseserver243alpine_ed2257` | **155.78%** | 97.30% | **6475.8 MB** | 2450.7 MB | 81 |
| `be23006c97e341e48800398ae2a91e83_postgres16alpine_890b3d` | **3.03%** | 0.54% | **32.4 MB** | 29.9 MB | 81 |
| `dt-redpanda` | **9.00%** | 2.02% | **911.0 MB** | 839.2 MB | 81 |
| `server-process` (Go Ingest/Consumer) | **35.30%** | 27.14% | **122.0 MB** | 87.9 MB | 81 |

> [!NOTE]
> Continuous background sampling captured every 2s during sustained ingestion load.
> **Process Sizing Disclaimer:** The server binary (`bin/server`) and the load generator (`bin/loadtest`) run directly on the host runner and share the runner's 2–4 vCPUs. The load generator's CPU footprint is co-located on the same box, so host resource readings are indicative of shared test-runner overhead, not dedicated production VM sizing.

