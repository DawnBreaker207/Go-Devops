import type { PagedData } from './api';

/** Customer "Account" screen types, mirroring /users/me/* endpoints spread across 3 Go DTO files (user, payment, auth); grouped here because one screen serves them all. */

/** No email: backend rejects email changes here, but still returns the full UserResponse. */
export interface UpdateProfilePayload {
  full_name: string;
  phone?: string;
}

/** Password re-entry required; wrong password is 401, remaining CONFIRMED tickets block with 409. */
export interface DeleteAccountPayload {
  password: string;
}

/** Finance view (paid/refunded how much, via which provider), unlike the ticket/booking view of GET /orders. */
export interface Transaction {
  payment_id: string;
  provider: string;
  txn_ref: string;
  status: 'pending' | 'paid' | 'failed' | 'refund_pending' | 'refunded';
  status_reason?: string;
  /** Expected amount, int64 whole VND. */
  amount: number;
  /** Actually received; present only after payment. */
  paid_amount?: number;
  paid_at?: string;
  refunded_at?: string;
  created_at: string;
  booking_id: string;
  showtime_id: string;
  movie_title: string;
  hall_name: string;
  start_at: string;
}

export type TransactionPage = PagedData<Transaction>;

/** Both flags false is valid; no "at least one" constraint. */
export interface NotificationPreference {
  booking_reminders: boolean;
  promo_offers: boolean;
}

export type UpdateNotificationPreferencePayload = NotificationPreference;

/** `is_current` is set only when the client sends its own device_id; see getDeviceId in auth.api. */
export interface Session {
  id: string;
  user_agent?: string;
  last_used_at?: string;
  created_at: string;
  is_current: boolean;
}
