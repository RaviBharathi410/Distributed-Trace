CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    name TEXT,
    key_hash TEXT NOT NULL,
    prefix TEXT NOT NULL,
    created_by UUID,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_org_id ON api_keys(org_id);
