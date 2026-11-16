# DistributedTrace Empirical Ingestion & Storage Benchmark Report

**Date:** 2026-11-16T10:06:49Z  
**Hardware Environment:** `Shared 2-4 vCPU box (GitHub Actions Runner / linux amd64) - INDICATIVE ONLY, NOT A SIZING GUIDE`  
**Dataset:** 1000000 spans sent (1000000 persisted) in 8192-span batches  
**Sustained Throughput:** `6051 spans/second` (duration: `165.26s`)  

## 1. ClickHouse Partition Consolidation (system.parts)

| Partition | Active Parts (Post-OPTIMIZE) | Rows | Bytes on Disk | Single Part Verified |
|---|---|---|---|---|
| `20261116` | 1 | 1000000 | **171994176 B** | ✅ Yes (1 part) |

## 2. Multi-Table Storage Footprint & Classification

| Table | Role / Classification | Active Parts | Rows | Compressed Data | Bytes on Disk |
|---|---|---|---|---|---|
| `otel_spans` | Primary Ingest (Raw Spans + Indexes) | 1 | 1000000 | 171986956 B | **171994176 B** |
| `service_metrics_hourly` | Auxiliary Aggregate Rollup (SummingMergeTree) | 0 | 0 | 0 B | **0 B** |
| `anomalies` | Auxiliary Incident Alerts (MergeTree) | 3 | 18243 | 1395081 B | **1398709 B** |

> [!NOTE]
> Telemetry ingest storage is accounted for primarily by `otel_spans` (columns + skipping indices + marks). The `anomalies` table is populated asynchronously by the Go server's background anomaly detector evaluating spans against statistical thresholds; with an 8% injected error rate and latency jitter, the detector recorded ~1.8% of spans as anomalies (~1.4 B/span disk footprint). `service_metrics_hourly` is an analytical rollup table (0 B during raw ingest).

## 3. Data Skipping Indices (Bloom Filter)

| Table | Index Name | Type | Expression | Granularity |
|---|---|---|---|---|
| `otel_spans` | `idx_service` | `bloom_filter` | `service_name` | 4 |

## 4. Top Column Compressed Footprint (otel_spans)

| Column | Compressed Bytes | Uncompressed Bytes | Compression Ratio |
|---|---|---|---|
| `tags` | 94196645 B | 580766800 B | 6.2x |
| `trace_id` | 33111114 B | 33000000 B | 1.0x |
| `parent_span_id` | 17062268 B | 17000000 B | 1.0x |
| `span_id` | 17061713 B | 17000000 B | 1.0x |
| `start_time` | 4042583 B | 8000000 B | 2.0x |
| `duration_ms` | 2936748 B | 4000000 B | 1.4x |
| `operation_name` | 2632013 B | 21240171 B | 8.1x |
| `error_message` | 437696 B | 6036574 B | 13.8x |
| `status_code` | 278517 B | 1000000 B | 3.6x |
| `org_id` | 153998 B | 37000000 B | 240.3x |

### 4.1 Tags Column Codec Empirical Comparison (1M Spans: LZ4 vs ZSTD(1) vs ZSTD(3))

| Codec | Compressed Bytes | Uncompressed Bytes | Compression Ratio | Tags B/Span | Compression Time (1M Spans) | Storage Savings vs LZ4 |
|---|---|---|---|---|---|---|
| `LZ4 (Default Baseline)` | 157228881 B | 580766800 B | 3.69x | 157.23 B | 3716 ms | Baseline (0.0%) |
| `ZSTD(1)` | 95225811 B | 580766800 B | 6.10x | 95.23 B | 4265 ms | ✅ **-39.4%** |
| `ZSTD(3)` | 94196645 B | 580766800 B | 6.17x | 94.20 B | 5156 ms | ✅ **-40.1%** |

> [!NOTE]
> Empirical codec comparison executed directly on the 1,000,000 live ingested span tags dataset in ClickHouse, isolating data density vs compression CPU throughput.


## 5. Empirical SaaS Cost-Per-Million-Spans Economics

- **Physical Disk Footprint per Span:** `173.39 bytes` (columns + skipping indices + marks)
- **Physical Disk Footprint per 1M Spans:** `0.161 GB`
- **Monthly Storage Cost (AWS EBS gp3):** `$0.0129 / million spans / month` ($0.08/GB-mo)
- **Monthly Storage Cost (AWS S3 Tiered):** `$0.0037 / million spans / month` ($0.023/GB-mo)
- **One-Time Network Wire Ingress:** `$0.00142 / million spans` (0.142 GB wire payload at $0.01/GB)
- **Initial Month Combined Unit Cost (EBS + Ingress):** `~$0.01434 / million spans`
- **Initial Month Combined Unit Cost (S3 Tiered + Ingress):** `~$0.00514 / million spans`

> [!NOTE]
> Recurring monthly physical storage fees and one-time cross-AZ network wire ingress costs are distinct economic line items and are accounted for separately above.

> [!IMPORTANT]
> **Hardware Disclaimer & Pricing Guardrail:** Results are indicative of GitHub runner hardware (shared 2-4 vCPU box). Final SaaS pricing tiers are strictly held pending dedicated cloud VM production sizing.

## 6. Baseline Query Latency & Deterministic I/O (20000000 Rows Scale)

### 6.1 Query Performance & Deterministic I/O Summary (20 Iterations)

| Query Operation | Scale | First Run (Cache Dropped) | P50 Latency | P95 Latency | Cold Read Rows | Avg Read Rows | Cold Read Bytes | Avg Read Bytes | Result Rows | Status / Analysis |
|---|---|---|---|---|---|---|---|---|---|---|
| `GetTraceWithSpans (Point Lookup)` | 20M rows | **243 ms** | **221.0 ms** | **261.0 ms** | 20000000 | 20000000 | 1723945536 B | 1724066802 B | 4 (avg 4.0) | ⚠️ Scans full partition (no bloom filter index on trace_id yet) |
| `SearchTraces (Single Service, Primary Key Pruned)` | 20M rows | **4 ms** | **7.0 ms** | **46.0 ms** | 8192 | 13797 | 664312 B | 1452698 B | 0 (avg 23.7) | ✅ Primary key pruned via (org_id, service_name, start_time) |
| `SearchTraces (All Services, Last 15 Min)` | 20M rows | **192 ms** | **7.0 ms** | **142.0 ms** | 1007616 | 170836 | 148077330 B | 16223698 B | 50 (avg 7.9) | ⚠️ Cross-service scan over 15 min window (no service filter) |
| `GetServiceGraph (Self-Join)` | 20M rows | **23 ms** | **6.0 ms** | **39.0 ms** | 1007616 | 170836 | 81910724 B | 11223288 B | 0 (avg 0.0) | ⚠️ Self-join scan across 15 min fixed window |

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
        Condition: (org_id in ['29972b2a-6879-4947-a353-dce7dcdbdf33', '29972b2a-6879-4947-a353-dce7dcdbdf33'])
        Parts: 3/3
        Granules: 2442/2442
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
                  Condition: and((start_time in (-Inf, '1794791400']), and((start_time in ['1794790800', +Inf)), and((service_name in ['service-0', 'service-0']), (org_id in ['29972b2a-6879-4947-a353-dce7dcdbdf33', '29972b2a-6879-4947-a353-dce7dcdbdf33']))))
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
                  Condition: and((start_time in (-Inf, '1794791700']), and((start_time in ['1794790800', +Inf)), (org_id in ['29972b2a-6879-4947-a353-dce7dcdbdf33', '29972b2a-6879-4947-a353-dce7dcdbdf33'])))
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
                Condition: and((start_time in (-Inf, '1794791700']), and((start_time in ['1794790800', +Inf)), (org_id in ['29972b2a-6879-4947-a353-dce7dcdbdf33', '29972b2a-6879-4947-a353-dce7dcdbdf33'])))
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
| `394faf0ff8834525a9d9c25842280978_clickhouseclickhouseserver243alpine_9a2d28` | **386.25%** | 110.33% | **5443.6 MB** | 2558.1 MB | 91 |
| `b448dc639b27498e92bb19cfab098902_postgres16alpine_39766f` | **3.63%** | 0.49% | **30.8 MB** | 30.1 MB | 91 |
| `dt-redpanda` | **20.15%** | 2.17% | **910.6 MB** | 844.2 MB | 91 |
| `server-process` (Go Ingest/Consumer) | **38.50%** | 28.65% | **114.5 MB** | 97.8 MB | 90 |

> [!NOTE]
> Continuous background sampling captured every 2s during sustained ingestion load.
> **Process Sizing Disclaimer:** The server binary (`bin/server`) and the load generator (`bin/loadtest`) run directly on the host runner and share the runner's 2–4 vCPUs. The load generator's CPU footprint is co-located on the same box, so host resource readings are indicative of shared test-runner overhead, not dedicated production VM sizing.

