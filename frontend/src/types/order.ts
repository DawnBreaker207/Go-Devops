import type { SeatType } from './hall';

// Mirrors Go order DTO.

export interface HoldPayload {
  show_id: string;
  seat_ids: string[];
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
  expires_at: string;
  seats: HeldSeat[];
  replaced_booking_id?: string;
}

export interface InitResult {
  booking_id: string;
  showtime_id: string;
  expires_at: string;
  ttl_seconds: number;
  reused?: boolean;
}

export interface RefreshResult {
  booking_id: string;
  expires_at: string;
}

export interface PayPayload {
  provider?: string;
}

export interface PayResult {
  payment_id: string;
  provider: string;
  txn_ref: string;
  redirect_url: string;
  expires_at?: string;
}

export interface PaymentProvider {
  name: string;
  display_name: string;
  default: boolean;
}
