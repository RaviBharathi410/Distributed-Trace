CREATE TABLE IF NOT EXISTS team_invites (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    invited_by UUID NOT NULL,
    accepted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    token TEXT UNIQUE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_team_invites_org_id ON team_invites(org_id);
CREATE INDEX IF NOT EXISTS idx_team_invites_token ON team_invites(token);
