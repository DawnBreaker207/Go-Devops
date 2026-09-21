import type { PageQuery } from './api';

/** Mirror of internal/dto/audit.go. Admin only. */

export type AuditOutcome = 'success' | 'failure';

/** `booking_id` threads one order's lifecycle (hold/pay/webhook/refund/redeem) across differing resource_type/resource_id. */
export interface AuditLog {
  id: string;
  actor_id?: string;
  actor_role?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  booking_id?: string;
  before_json?: Record<string, unknown>;
  after_json?: Record<string, unknown>;
  ip?: string;
  user_agent?: string;
  outcome: AuditOutcome;
  error_message?: string;
  created_at: string;
}

/** from/to bound created_at (RFC3339 or YYYY-MM-DD), same convention as the daily report. */
export interface AuditLogListQuery extends PageQuery {
  action?: string;
  resource_type?: string;
  resource_id?: string;
  booking_id?: string;
  actor_id?: string;
  outcome?: AuditOutcome;
  from?: string;
  to?: string;
}
