import type { ColSpan, ScreenPosition, SeatType } from './hall';
import type { MovieAgeRating } from './movie';
import type { ShowtimeStatus } from './showtime';

/** Mirror of dto.SeatMapResponse. Plain-JWT guard (`protected`, no RequireRoles), so admin/staff can call it; but POST /orders/hold is customer-only. Gate: onSale() folds three conditions (show `open`, start_at in future, movie `showing`) into one 404 "showtime is not open"; a just-started show 404s, not 409. Both 404s share code 40400 and differ by English `message` only. */

/** No request carries seat status, so no binding tag. */
export type SeatStatus = 'available' | 'held' | 'sold';

export interface SeatMapSeat {
  /** Physical hall seat, stable across shows. Never send this id to /orders/hold. */
  id: string;
  /** The only omitempty field here, and what /orders/hold actually wants. Absent (not null/"") when no showtime_seats row exists (LEFT JOIN); such seats still report `status: 'available'`, so treat `!showtime_seat_id` as unpickable. */
  showtime_seat_id?: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  /** Gaps are still returned with 'available' and an id; holding one fails 400/40001 ErrSeatNotSellable. */
  is_gap: boolean;
  col_span: ColSpan;
  status: SeatStatus;
  /** int64 whole VND. 0 = hall price missing for this seat type, not free; holding it fails 409/40900 ErrMissingHallPrice. */
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
  /** SHOWTIME status, not seat. Always 'open': onSale() 404s everything else. */
  status: ShowtimeStatus;
  screen_position: ScreenPosition;
  /** Draft hint only; backend checks length (<=49), not values. */
  aisle_after_cols: number[];
  /** SPARSE: only seat types actually present in this hall. Present-but-unpriced types read 0. */
  prices: Partial<Record<SeatType, number>>;
  /** Pre-sorted by seats.row_index then col_number. KEEP this order. */
  seats: SeatMapSeat[];
}
