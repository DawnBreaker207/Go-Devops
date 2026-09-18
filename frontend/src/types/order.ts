import type { SeatType } from './hall';

// PaymentSummary / OrderShowtime / OrderStatus / OrderDetail / Ticket deu da
// duoc khai bao o './booking' cho man van hanh - cung DTO Go, nen dung lai chu
// khong khai lan hai.

/**
 * Luong dat ve cua KHACH - mirror cua internal/dto/booking.go.
 *
 * Quyen: ca nhanh /orders/* la RequireRoles(customer) TRU `GET /orders` (khong
 * co RequireRoles - moi role da dang nhap goi duoc, va no tra don CUA CHINH
 * nguoi goi). Admin/staff xem duoc so do ghe nhung khong dat ve duoc.
 *
 * Vong doi: hold -> pay -> (cong thanh toan) -> confirm.
 * - `hold` giu ghe trong `booking.hold_ttl_minutes` phut (mac dinh 10, tran 60).
 * - `pay` tra ve `redirect_url` cua cong thanh toan; so tien LUON lay tu don,
 *   khong bao gio tu client.
 * - `confirm` goi `finalize`: chua tra tien thi loi ErrBookingNotPaid, da tra
 *   thi don thanh `confirmed` va tra ve ca danh sach ve.
 */

export interface HoldPayload {
  show_id: string;
  /** PHAI la showtime_seat_id, khong phai seats.id. Toi thieu 1. */
  seat_ids: string[];
  /** Tuy chon, toi da 128 ky tu. Gui lai cung khoa de tranh giu cho trung. */
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
  /** RFC3339. Het han thi ghe duoc nha ra va don chuyen sang `expired`. */
  expires_at: string;
  seats: HeldSeat[];
  /** omitempty: chi co khi lan giu nay THAY THE mot lan giu truoc do. */
  replaced_booking_id?: string;
}

/** Bo trong `provider` thi backend dung cong mac dinh (xem GET /payments/providers). */
export interface PayPayload {
  provider?: string;
}

export interface PayResult {
  payment_id: string;
  provider: string;
  txn_ref: string;
  /** Dua trinh duyet toi day. Voi cong `mock` day la trang gia lap trong
   *  chinh backend (engine.Any(sim.SimulatorPath())). */
  redirect_url: string;
  expires_at?: string;
}

export interface PaymentProvider {
  name: string;
  display_name: string;
  default: boolean;
}
