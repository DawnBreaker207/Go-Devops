import type { PageQuery } from './api';

/** Mirror of models.BatchJob and the job registrations in cmd/server/main.go. Both endpoints are admin-only. */

export type BatchJobStatus = 'running' | 'success' | 'failed' | 'skipped' | 'stopped';
export type BatchTriggeredBy = 'cron' | 'manual' | 'confirm';

/** Registered job names; drives the "Run now" button. */
export const BATCH_JOB_NAMES = [
  'closeDay',
  'sweepExpiredHolds',
  'sendTicketEmails',
  'cleanup',
] as const;
export type BatchJobName = (typeof BATCH_JOB_NAMES)[number];

/** One job-run history row. */
export interface BatchJob {
  id: string;
  job_name: string;
  triggered_by: BatchTriggeredBy;
  status: BatchJobStatus;
  processed_rows: number;
  skipped_rows: number;
  error_message?: string;
  started_at: string;
  finished_at?: string;
  created_at: string;
}

/** PageQuery.search filters by job_name. */
export type BatchJobListQuery = PageQuery;

/** POST .../:name/run result (202 Accepted). */
export interface BatchRunResult {
  job: string;
  run_id: string;
  status: BatchJobStatus;
  triggered_by: BatchTriggeredBy;
}
