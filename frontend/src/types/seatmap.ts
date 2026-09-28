import type { ColSpan, ScreenPosition, SeatType } from './hall';
import type { MovieAgeRating } from './movie';
import type { ShowtimeStatus } from './showtime';

// Mirrors Go DTO dto.SeatMapResponse.
export type SeatStatus = 'available' | 'held' | 'sold';

export interface SeatMapSeat {
  id: string;
  showtime_seat_id?: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  is_gap: boolean;
  col_span: ColSpan;
  status: SeatStatus;
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
  status: ShowtimeStatus;
  screen_position: ScreenPosition;
  aisle_after_cols: number[];
  prices: Partial<Record<SeatType, number>>;
  seats: SeatMapSeat[];
}
