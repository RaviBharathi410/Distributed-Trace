CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT 'self-hosted',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
