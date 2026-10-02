CREATE TABLE IF NOT EXISTS tenant_llm_costs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id VARCHAR(64) NOT NULL,
    anomaly_id VARCHAR(64) NOT NULL,
    model VARCHAR(64) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    estimated_cost_usd DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_llm_costs_org ON tenant_llm_costs (org_id, created_at);
