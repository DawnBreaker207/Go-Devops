import type { MovieAgeRating } from './movie';
import type { PageQuery } from './api';

/** Every value the column can hold. `cancelled` was added by migration 000007
 *  (`ck_showtime_status`) and is reachable ONLY through
 *  `POST /admin/showtimes/:id/cancel`, which also refunds. */
export type ShowtimeStatus = 'open' | 'closed' | 'cancelled';

/** The statuses a form may SET. Deliberately excludes `cancelled`: the full-replace
 *  PUT would happily write it, and that would mark a showtime cancelled while
 *  leaving every paid booking untouched — cancelling has to go through the
 *  endpoint that moves the money. Use SHOWTIME_STATUSES for selects and filters,
 *  ShowtimeStatus for anything that only DISPLAYS a status. */
export const SHOWTIME_STATUSES: ShowtimeStatus[] = ['open', 'closed'];

/** No field is omitempty so keys are always present, but values aren't: admin GETs are complete (+07:00); POST/PUT return empty movie_title/age_rating/hall_name and zero created_at/updated_at. Never put POST/PUT results straight into the table; invalidate and refetch or the edited row loses its names. */
export interface Showtime {
  id: string;
  movie_id: string;
  movie_title: string;
  age_rating: MovieAgeRating | '';
  hall_id: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  status: ShowtimeStatus;
  created_at: string;
  updated_at: string;
}

/** Same body for POST and PUT. movie_id/hall_id/start_at are `required` on BOTH verbs: even a status-only change resends all three. No end_at; backend derives it from movie duration. status: POST accepts then ignores (always 'open'); PUT empty keeps current. */
export interface ShowtimePayload {
  movie_id: string;
  hall_id: string;
  start_at: string;
  /** Never 'cancelled' — see SHOWTIME_STATUSES. */
  status?: 'open' | 'closed';
}

export interface ShowtimeListQuery extends PageQuery {
  movie_id?: string;
  hall_id?: string;
  status?: ShowtimeStatus;
  /** YYYY-MM-DD. Wins over from/to; don't send together. */
  date?: string;
  from?: string;
  /** YYYY-MM-DD, inclusive. */
  to?: string;
  sort?: 'start_at' | 'created_at';
  order?: 'asc' | 'desc';
}

/** Customer picker shape (has `from_price`, no created_at/updated_at). Public bare array. Clamped to ONE day (today in cinema time without `date`); implicit filters: movie `showing`, show `open`, start_at >= now, hall with all 4 prices. Absent != nonexistent. Non-`showing` movies return [] not 404. */
export interface ShowtimeListItem {
  id: string;
  movie_id: string;
  movie_title: string;
  age_rating: string;
  hall_id: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  status: ShowtimeStatus;
  /** omitempty: absent when 0, meaning the hall has no prices. */
  from_price?: number;
}

/** POST /admin/showtimes/:id/cancel. `bookings_affected` counts the bookings the
 *  cascade touched: confirmed ones flip straight to refunded with their tickets
 *  voided, pending ones are released and refunded too if they had already paid. */
export interface ShowtimeCancelResult {
  showtime_id: string;
  status: ShowtimeStatus;
  bookings_affected: number;
}
