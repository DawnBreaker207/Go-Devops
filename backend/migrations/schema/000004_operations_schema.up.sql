-- Operations schema (platform + Track 5): activity trail, batch job runs and
-- daily revenue rollups. outcome marks whether an audited attempt succeeded
-- (written in the business transaction) or failed (written after the fact by
-- the audit middleware, outside any transaction — nothing was changed).

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

CREATE TABLE IF NOT EXISTS batch_jobs (
    id             UUID PRIMARY KEY,
    job_name       VARCHAR(128) NOT NULL,
    triggered_by   VARCHAR(32)  NOT NULL DEFAULT 'cron',
    status         VARCHAR(16)  NOT NULL DEFAULT 'running',
    processed_rows INTEGER      NOT NULL DEFAULT 0,
    skipped_rows   INTEGER      NOT NULL DEFAULT 0,
    error_message  TEXT,
    started_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_batch_job_status CHECK (status IN ('running','success','failed','skipped','stopped'))
);

CREATE INDEX IF NOT EXISTS idx_batch_jobs_name_started ON batch_jobs (job_name, started_at);

-- At most one RUNNING run per job at any time.
CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_jobs_one_running
    ON batch_jobs (job_name) WHERE status = 'running';

CREATE TABLE IF NOT EXISTS daily_aggregates (
    id             UUID PRIMARY KEY,
    report_date    DATE NOT NULL,
    total_revenue  BIGINT       NOT NULL DEFAULT 0,
    tickets_sold   INTEGER      NOT NULL DEFAULT 0,
    seats_sold     INTEGER      NOT NULL DEFAULT 0,
    capacity       INTEGER      NOT NULL DEFAULT 0,
    occupancy_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    breakdown      JSONB        NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_daily_aggregates_report_date UNIQUE (report_date)
);