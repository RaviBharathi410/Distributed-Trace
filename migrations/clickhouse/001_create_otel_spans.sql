CREATE TABLE IF NOT EXISTS otel_spans (
    org_id String,
    trace_id String,
    span_id String,
    parent_span_id String,
    service_name String,
    operation_name String,
    duration_ms UInt32,
    status_code UInt8,
    error_message String,
    tags Map(String, String),
    start_time DateTime64(3, 'UTC'),
    INDEX idx_service service_name TYPE bloom_filter GRANULARITY 4
) Engine = MergeTree()
ORDER BY (org_id, service_name, start_time)
PARTITION BY toYYYYMMDD(start_time)
TTL toDateTime(start_time) + INTERVAL 14 DAY WHERE status_code != 2,
    toDateTime(start_time) + INTERVAL 90 DAY;
