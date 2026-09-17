DROP INDEX IF EXISTS uq_daily_aggregates_report_date_branch;
ALTER TABLE daily_aggregates DROP COLUMN IF EXISTS branch_id;
ALTER TABLE daily_aggregates ADD CONSTRAINT uq_daily_aggregates_report_date UNIQUE (report_date);

ALTER TABLE users DROP COLUMN IF EXISTS branch_id;

ALTER TABLE halls DROP COLUMN IF EXISTS branch_id;

DROP TABLE IF EXISTS branches;
