import type { SeatType } from './hall';

// PaymentSummary/OrderShowtime/OrderStatus/OrderDetail/Ticket are declared in './booking' (same Go DTOs); reuse, don't redeclare.

/** Customer booking flow. Auth: /orders/* requires customer EXCEPT GET /orders (any authenticated role, returns caller's own orders). Admin/staff can view seat maps but cannot book. Lifecycle: hold -> pay -> (gateway) -> confirm. hold keeps seats for booking.hold_ttl_minutes (default 10, max 60); pay returns redirect_url with the amount always from the order, never the client; confirm finalizes (unpaid is ErrBookingNotPaid, paid becomes `confirmed` with tickets). */

export interface HoldPayload {
  show_id: string;
  /** Must be showtime_seat_id, not seats.id. At least 1. */
  seat_ids: string[];
  /** Optional, max 128 chars. Resend the same key to avoid duplicate holds. */
  idempotency_key?: string;
}

export interface HeldSeat {
  showtime_seat_id: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  price: number;
}

export interface HoldResult {
  booking_id: string;
  showtime_id: string;
  total_amount: number;
  /** RFC3339. Expiry frees the seats and flips the order to `expired`. */
  expires_at: string;
  seats: HeldSeat[];
  /** omitempty: present only when this hold REPLACED a previous one. */
  replaced_booking_id?: string;
}

/** Opens an empty PENDING order to count down from entry. */
export interface InitResult {
  booking_id: string;
  showtime_id: string;
  /** RFC3339. */
  expires_at: string;
  /** Seconds for the client ticker; no time math needed. */
  ttl_seconds: number;
  /** True when returning an existing valid pending order instead of creating one. */
  reused?: boolean;
}

/** POST /orders/:id/refresh - heartbeat gian han trong tran lifetime. */
export interface RefreshResult {
  booking_id: string;
  /** RFC3339. */
  expires_at: string;
}

/** Empty `provider` uses the default gateway (see GET /payments/providers). */
export interface PayPayload {
  provider?: string;
}

export interface PayResult {
  payment_id: string;
  provider: string;
  txn_ref: string;
  /** Send the browser here. For the `mock` provider this is the in-backend simulator page. */
  redirect_url: string;
  expires_at?: string;
}

export interface PaymentProvider {
  name: string;
  display_name: string;
  default: boolean;
}
