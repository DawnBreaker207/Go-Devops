-- Module multi-branch: scopes halls/staff/vouchers to a branch; existing
-- rows backfill to one default branch.

CREATE TABLE IF NOT EXISTS branches (
    id         UUID         PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    address    VARCHAR(500),
    active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_branches_name ON branches (name);

INSERT INTO branches (id, name, address, active)
VALUES ('00000000-0000-0000-0000-000000000001', 'Chi nhanh chinh', NULL, true)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE halls ADD COLUMN IF NOT EXISTS branch_id UUID REFERENCES branches(id);
UPDATE halls SET branch_id = '00000000-0000-0000-0000-000000000001' WHERE branch_id IS NULL;
ALTER TABLE halls ALTER COLUMN branch_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_halls_branch ON halls (branch_id);

-- Nullable: an unassigned staff account serves every branch.
ALTER TABLE users ADD COLUMN IF NOT EXISTS branch_id UUID REFERENCES branches(id);
CREATE INDEX IF NOT EXISTS idx_users_branch ON users (branch_id) WHERE branch_id IS NOT NULL;

-- Reserved for a future per-branch closeDay rework; always NULL for now.
ALTER TABLE daily_aggregates DROP CONSTRAINT IF EXISTS uq_daily_aggregates_report_date;
ALTER TABLE daily_aggregates ADD COLUMN IF NOT EXISTS branch_id UUID REFERENCES branches(id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_daily_aggregates_report_date_branch
    ON daily_aggregates (report_date, branch_id) NULLS NOT DISTINCT;
