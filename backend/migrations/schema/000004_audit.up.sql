-- Module audit: activity trail (F13). outcome marks whether an audited attempt
-- succeeded (written in the business transaction) or failed (written after the
-- fact by the audit middleware, outside any transaction — nothing was changed).

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID PRIMARY KEY,
    actor_id      UUID,
    actor_role    VARCHAR(32),
    action        VARCHAR(64)  NOT NULL,
    resource_type VARCHAR(64)  NOT NULL,
    resource_id   VARCHAR(128),
    before_json   JSONB,
    after_json    JSONB,
    ip            VARCHAR(64),
    user_agent    VARCHAR(512),
    outcome       VARCHAR(16)  NOT NULL DEFAULT 'success',
    error_message TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_audit_outcome CHECK (outcome IN ('success','failure'))
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs (actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);
