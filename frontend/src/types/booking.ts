import type { PageQuery } from './api';
// SeatType belongs to the hall domain; defined in './hall' so the barrel doesn't export one name twice.
import type { SeatType } from './hall';

export type BookingStatus = 'pending' | 'confirmed' | 'expired' | 'refunded';
export type PaymentStatus = 'pending' | 'paid' | 'failed' | 'refund_pending' | 'refunded';
export type SoldVia = 'online' | 'counter';
export type TicketStatus = 'issued' | 'redeemed' | 'void';

export const BOOKING_STATUSES: BookingStatus[] = ['pending', 'confirmed', 'expired', 'refunded'];
export const PAYMENT_STATUSES: PaymentStatus[] = [
  'pending',
  'paid',
  'failed',
  'refund_pending',
  'refunded',
];
export const SOLD_VIA: SoldVia[] = ['online', 'counter'];

/** Why an order left its normal state. */
export type BookingStatusReason =
  | 'replaced'
  | 'hold_expired'
  | 'seats_lost'
  | 'showtime_closed'
  | 'amount_mismatch'
  | 'paid_after_expiry'
  | 'canceled'
  | 'showtime_cancelled';

export type PaymentStatusReason = 'create_failed' | 'declined' | 'abandoned' | 'duplicate_payment';

export interface PaymentSummary {
  id: string;
  provider: string;
  txn_ref: string;
  status: PaymentStatus;
  status_reason?: PaymentStatusReason;
  amount: number;
  paid_amount?: number;
  paid_at?: string;
  refunded_at?: string;
}

export interface OrderShowtime {
  movie_id: string;
  movie_title: string;
  age_rating: string;
  hall_id: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  /** Computed at response time via time.Now(), not a DB column. */
  started: boolean;
  ended: boolean;
}

/** Shared core of an order. */
export interface OrderStatus {
  id: string;
  showtime_id: string;
  status: BookingStatus;
  status_reason?: BookingStatusReason;
  total_amount: number;
  discount_amount: number;
  payable_amount: number;
  created_at: string;
  expires_at?: string;
  paid_at?: string;
  /** The attempt currently holding money. Absent for unpaid holds (normal). */
  payment?: PaymentSummary;
  showtime?: OrderShowtime;
}

export interface OrderCustomer {
  user_id?: string;
  email?: string;
  full_name?: string;
  phone?: string;
}

export interface AdminOrder extends OrderStatus {
  sold_via: SoldVia;
  seats: number;
  customer?: OrderCustomer;
}

export interface Ticket {
  id: string;
  showtime_seat_id: string;
  seat_label: string;
  seat_type: SeatType;
  price: number;
  code: string;
  status: TicketStatus;
}

export interface TicketQR {
  ticket_id: string;
  code: string;
  qr_base64: string;
}

export interface OrderDetail extends OrderStatus {
  tickets: Ticket[];
}

export interface AdminOrderListQuery extends PageQuery {
  status?: BookingStatus;
  payment_status?: PaymentStatus;
  sold_via?: SoldVia;
  showtime_id?: string;
  movie_id?: string;
  user_id?: string;
  date?: string;
  from?: string;
  to?: string;
  sort?: 'created_at' | 'paid_at' | 'total_amount' | 'start_at';
  order?: 'asc' | 'desc';
}

export interface RedeemPayload {
  showtime_id: string;
}

export type RedeemStatus = 'ok' | 'used' | 'wrong_show' | 'not_found' | 'too_early' | 'closed';

export interface RedeemResult {
  status: RedeemStatus;
  ticket_id?: string;
  showtime_id?: string;
  movie_title?: string;
  age_rating?: string;
  hall_name?: string;
  seat_label?: string;
  start_at?: string;
  checkin_opens_at?: string;
  checkin_closes_at?: string;
}
