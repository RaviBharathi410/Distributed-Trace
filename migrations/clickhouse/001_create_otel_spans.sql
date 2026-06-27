CREATE TABLE IF NOT EXISTS otel_spans (
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
ORDER BY (service_name, start_time)
PARTITION BY toYYYYMMDD(start_time)
TTL start_time + INTERVAL 30 DAY;
