-- Module batch & report: batch job runs and daily revenue rollups (F17, F21).
-- Finished runs are removed by the cleanup job after the audit retention.

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
    CONSTRAINT ck_batch_job_status CHECK (status IN ('running','success','failed','skipped','stopped')),
    CONSTRAINT ck_batch_job_trigger CHECK (triggered_by IN ('cron','manual','confirm'))
);

CREATE INDEX IF NOT EXISTS idx_batch_jobs_name_started ON batch_jobs (job_name, started_at);
-- Admin run log (newest first) and the cleanup job.
CREATE INDEX IF NOT EXISTS idx_batch_jobs_started ON batch_jobs (started_at);

-- At most one RUNNING run per job at any time.
CREATE UNIQUE INDEX IF NOT EXISTS uq_batch_jobs_one_running
    ON batch_jobs (job_name) WHERE status = 'running';

CREATE TABLE IF NOT EXISTS daily_aggregates (
    id             UUID PRIMARY KEY,
    report_date    DATE         NOT NULL,
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
