import type { ColSpan, ScreenPosition, SeatType } from './hall';
import type { MovieAgeRating } from './movie';
import type { ShowtimeStatus } from './showtime';

/**
 * Mirror cua dto.SeatMapResponse - GET /api/v1/shows/:id/seats.
 *
 * Guard la JWT THUAN (nhom `protected`, khong RequireRoles), nen admin/staff
 * cung goi duoc; nhung `POST /orders/hold` thi la RequireRoles(customer), nen
 * chi khach moi giu duoc ghe.
 *
 * CONG GATE: `onSale()` gop ba dieu kien vao MOT loi 404 "showtime is not open"
 * - suat phai `open`, `start_at` phai con o tuong lai, va phim phai `showing`.
 * Mot suat vua bat dau se tra 404 chu khong phai 409. Hai loai 404 cua endpoint
 * nay ("showtime not found" va "showtime is not open") dung CHUNG ma 40400 va
 * chi phan biet duoc bang `message` tieng Anh.
 */

/** models.SeatStatus* + SQL CHECK ck_showtime_seat_status. Khong co binding tag
 *  vi khong request nao mang trang thai ghe len. */
export type SeatStatus = 'available' | 'held' | 'sold';

export interface SeatMapSeat {
  /** seats.id - ghe VAT LY cua phong, khong doi qua cac suat.
   *  TUYET DOI khong gui id nay len /orders/hold. */
  id: string;
  /**
   * showtime_seats.id - field DUY NHAT co omitempty tren struct nay, va la thu
   * ma /orders/hold thuc su can.
   *
   * No VANG MAT (khong phai null, khong phai "") khi ghe chua co ban ghi
   * showtime_seats, vi SQL la LEFT JOIN. Ghe do van hien `status: 'available'`
   * (service doi "" thanh available), tuc TRONG NHU DAT DUOC ma khong dat duoc.
   * Coi `!showtime_seat_id` la khong chon duoc.
   */
  showtime_seat_id?: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  /** Ghe gap VAN duoc tra ve, VAN co status 'available' va VAN co
   *  showtime_seat_id. Giu no la 400/40001 ErrSeatNotSellable. */
  is_gap: boolean;
  col_span: ColSpan;
  status: SeatStatus;
  /** int64 VND nguyen. 0 = phong CHUA cau hinh gia cho loai ghe nay, khong phai
   *  mien phi. Giu ghe do la 409/40900 ErrMissingHallPrice. */
  price: number;
}

export interface SeatMap {
  showtime_id: string;
  movie_id: string;
  movie_title: string;
  age_rating: MovieAgeRating;
  hall_id: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  /** Trang thai cua SUAT CHIEU, khong phai cua ghe. Luon la 'open' vi onSale()
   *  da 404 moi truong hop khac. */
  status: ShowtimeStatus;
  screen_position: ScreenPosition;
  /** Chi la goi y ve; backend kiem do dai (<=49) chu khong kiem gia tri. */
  aisle_after_cols: number[];
  /** MOT PHAN: chi chua cac loai ghe THUC SU co trong phong nay. Loai co ghe
   *  nhung chua co gia thi co mat voi gia tri 0. */
  prices: Partial<Record<SeatType, number>>;
  /** Da sap theo seats.row_index roi col_number. GIU NGUYEN thu tu nay. */
  seats: SeatMapSeat[];
}
