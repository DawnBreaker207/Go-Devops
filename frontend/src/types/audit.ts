import type { PageQuery } from './api';

// Mirrors Go DTO internal/dto/audit.go.

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

export const AUDIT_RESOURCE_TYPES = [
  'article',
  'batch_job',
  'booking',
  'campaign',
  'combo_order',
  'concession',
  'discount',
  'hall',
  'media',
  'movie',
  'pricing_rule',
  'seat',
  'seat_base_price',
  'session',
  'showtime',
  'ticket',
  'user',
] as const;

export const AUDIT_ACTIONS = [
  'admin.add_hall_column',
  'admin.add_hall_row',
  'admin.attach_campaign_article',
  'admin.attach_campaign_combo',
  'admin.attach_campaign_discount',
  'admin.bulk_update_seats',
  'admin.cancel_showtime',
  'admin.clone_hall',
  'admin.create_article',
  'admin.create_campaign',
  'admin.create_concession',
  'admin.create_discount',
  'admin.create_hall',
  'admin.create_movie',
  'admin.create_pricing_rule',
  'admin.create_showtime',
  'admin.create_user',
  'admin.delete_article',
  'admin.delete_campaign',
  'admin.delete_concession',
  'admin.delete_discount',
  'admin.delete_hall',
  'admin.delete_hall_row',
  'admin.delete_movie',
  'admin.delete_pricing_rule',
  'admin.delete_showtime',
  'admin.detach_campaign_article',
  'admin.detach_campaign_combo',
  'admin.detach_campaign_discount',
  'admin.merge_hall_seats',
  'admin.run_job',
  'admin.set_base_price',
  'admin.split_hall_seat',
  'admin.update_article',
  'admin.update_campaign',
  'admin.update_concession',
  'admin.update_discount',
  'admin.update_hall',
  'admin.update_hall_layout',
  'admin.update_hall_seat',
  'admin.update_movie',
  'admin.update_pricing_rule',
  'admin.update_showtime',
  'admin.update_user',
  'admin.upload_poster',
  'auth.accept_terms',
  'auth.forgot_password',
  'auth.login',
  'auth.logout',
  'auth.refresh',
  'auth.register',
  'auth.reset_password',
  'combo_orders.create',
  'orders.apply_discount',
  'orders.cancel',
  'orders.confirm',
  'orders.counter_sell',
  'orders.hold',
  'orders.init',
  'orders.pay',
  'orders.refresh',
  'orders.remove_discount',
  'staff.redeem_ticket',
  'users.change_password',
  'users.delete_me',
  'users.revoke_session',
  'users.update_notification_preferences',
  'users.update_profile',
] as const;
