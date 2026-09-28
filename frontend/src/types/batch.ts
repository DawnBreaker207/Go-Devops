import type { PageQuery } from './api';

// Mirrors Go DTO models.BatchJob.

export type BatchJobStatus = 'running' | 'success' | 'failed' | 'skipped' | 'stopped';
export type BatchTriggeredBy = 'cron' | 'manual' | 'confirm';

export const BATCH_JOB_NAMES = [
  'closeDay',
  'sweepExpiredHolds',
  'sendTicketEmails',
  'cleanup',
] as const;
export type BatchJobName = (typeof BATCH_JOB_NAMES)[number];

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

export type BatchJobListQuery = PageQuery;

export interface BatchRunResult {
  job: string;
  run_id: string;
  status: BatchJobStatus;
  triggered_by: BatchTriggeredBy;
}
