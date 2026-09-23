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
  /** Operator-cancelled show (not a customer fault); order flips to `refunded` with automatic refund. */
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
  /** SEAT SUBTOTAL, before any discount. NOT the amount due - rendering this as
   *  "amount to pay" shows the wrong number on a discounted order. */
  total_amount: number;
  /** What a discount code took off. 0 when none is applied. */
  discount_amount: number;
  /** total_amount - discount_amount: what the gateway will actually charge.
   *  Always present (no omitempty on the backend), so it never silently reads 0. */
  payable_amount: number;
  created_at: string;
  expires_at?: string;
  paid_at?: string;
  /** The attempt currently holding money. Absent for unpaid holds (normal). */
  payment?: PaymentSummary;
  showtime?: OrderShowtime;
}

/** Online with account: { user_id, email, full_name, phone? }. Counter without account: { full_name?, phone? } only. Counter with neither has no object at all. */
export interface OrderCustomer {
  user_id?: string;
  email?: string;
  full_name?: string;
  phone?: string;
}

/** One row of GET /admin/orders. */
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

/** GET /tickets/:id/qr. `qr_base64` is PNG without the `data:image/...` prefix; prepend for `src`. Works after showtime. */
export interface TicketQR {
  ticket_id: string;
  code: string;
  qr_base64: string;
}

/** GET /staff/orders/:id. No customer/sold_via/seats (customer e-ticket shape); the admin drawer joins those from the opening table row. */
export interface OrderDetail extends OrderStatus {
  tickets: Ticket[];
}

export interface AdminOrderListQuery extends PageQuery {
  status?: BookingStatus;
  /** Matches held-payment orders only; open-checkout holds never match. */
  payment_status?: PaymentStatus;
  sold_via?: SoldVia;
  showtime_id?: string;
  movie_id?: string;
  user_id?: string;
  /** YYYY-MM-DD on created_at. Wins over from/to. */
  date?: string;
  from?: string;
  to?: string;
  sort?: 'created_at' | 'paid_at' | 'total_amount' | 'start_at';
  order?: 'asc' | 'desc';
}

/** POST /tickets/:id/redeem. `id` is a ticket id OR a QR code. */
export interface RedeemPayload {
  showtime_id: string;
}

/** Outside check-in window; see checkin_opens_at/checkin_closes_at. */
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
