CREATE TABLE IF NOT EXISTS service_metrics_hourly (
    org_id String,
    service_name String,
    hour DateTime,
    p50_ms Float64,
    p95_ms Float64,
    p99_ms Float64,
    error_rate Float64,
    request_count UInt64
) Engine = SummingMergeTree()
ORDER BY (org_id, service_name, hour)
TTL hour + INTERVAL 180 DAY;
