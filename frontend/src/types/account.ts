import type { PagedData } from './api';

// Mirrors Go DTO /users/me.
export interface UpdateProfilePayload {
  full_name: string;
  phone?: string;
}

export interface DeleteAccountPayload {
  password: string;
}

export interface Transaction {
  payment_id: string;
  provider: string;
  txn_ref: string;
  status: 'pending' | 'paid' | 'failed' | 'refund_pending' | 'refunded';
  status_reason?: string;
  amount: number;
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

export interface NotificationPreference {
  booking_reminders: boolean;
  promo_offers: boolean;
}

export type UpdateNotificationPreferencePayload = NotificationPreference;

export interface Session {
  id: string;
  user_agent?: string;
  last_used_at?: string;
  created_at: string;
  is_current: boolean;
}
