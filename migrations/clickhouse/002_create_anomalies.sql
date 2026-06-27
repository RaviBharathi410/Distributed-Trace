CREATE TABLE IF NOT EXISTS anomalies (
    id String,
    org_id String,
    service_name String,
    operation_name String,
    root_cause_service String,
    severity String,
    z_score Float64,
    baseline_latency_ms Float64,
    observed_latency_ms Float64,
    error_rate_delta Float64,
    status String DEFAULT 'active',
    detected_at DateTime64(3, 'UTC'),
    resolved_at Nullable(DateTime64(3, 'UTC')),
    root_cause_path Array(String)
) Engine = MergeTree()
ORDER BY (org_id, detected_at);
