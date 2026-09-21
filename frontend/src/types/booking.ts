import type { PageQuery } from './api';
// SeatType thuoc domain hall (models.AllSeatTypes); booking chi muon dung lai.
// Dinh nghia nam o './hall' de barrel khong export trung mot ten hai lan.
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

/** models.Reason* - ly do mot don roi khoi trang thai binh thuong. */
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
  /** Backend tinh theo time.Now() luc tra loi, khong phai cot trong DB. */
  started: boolean;
  ended: boolean;
}

/** dto.OrderStatusResponse - phan chung cua mot don. */
export interface OrderStatus {
  id: string;
  showtime_id: string;
  status: BookingStatus;
  status_reason?: BookingStatusReason;
  total_amount: number;
  created_at: string;
  expires_at?: string;
  paid_at?: string;
  /** Attempt ma don DANG GIU tien. Hold chua tra tien thi khong co. */
  payment?: PaymentSummary;
  showtime?: OrderShowtime;
}

/**
 * Don online co tai khoan: { user_id, email, full_name, phone? }.
 * Ban tai quay KHONG co tai khoan: chi { full_name?, phone? }.
 * Ban tai quay khong ghi ca ten lan sdt thi khong co object nay luon.
 */
export interface OrderCustomer {
  user_id?: string;
  email?: string;
  full_name?: string;
  phone?: string;
}

/** Mot dong cua GET /admin/orders. */
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

/**
 * GET /staff/orders/:id. KHONG co customer / sold_via / seats - day la hinh
 * dang ve dien tu cua khach. Drawer cua admin phai mang ba thu do sang tu dong
 * trong bang ma no duoc mo ra.
 */
export interface OrderDetail extends OrderStatus {
  tickets: Ticket[];
}

export interface AdminOrderListQuery extends PageQuery {
  status?: BookingStatus;
  /** Chi khop don DA GIU tien; hold con checkout mo thi khong bao gio khop. */
  payment_status?: PaymentStatus;
  sold_via?: SoldVia;
  showtime_id?: string;
  movie_id?: string;
  user_id?: string;
  /** YYYY-MM-DD tren created_at. Uu tien hon from/to. */
  date?: string;
  from?: string;
  to?: string;
  sort?: 'created_at' | 'paid_at' | 'total_amount' | 'start_at';
  order?: 'asc' | 'desc';
}
