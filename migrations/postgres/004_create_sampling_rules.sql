CREATE TABLE IF NOT EXISTS sampling_rules (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    rule_type TEXT NOT NULL, -- error, slow, new_route, always
    threshold_ms INTEGER NOT NULL DEFAULT 0,
    sample_rate NUMERIC(4, 3) NOT NULL DEFAULT 1.000,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sampling_rules_org_id ON sampling_rules(org_id);
