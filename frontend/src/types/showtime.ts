import type { MovieAgeRating } from './movie';
import type { PageQuery } from './api';

// Mirrors Go showtime DTO.
export type ShowtimeStatus = 'open' | 'closed' | 'cancelled';

export const SHOWTIME_STATUSES: ShowtimeStatus[] = ['open', 'closed'];

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

export interface ShowtimePayload {
  movie_id: string;
  hall_id: string;
  start_at: string;
  status?: 'open' | 'closed';
}

export interface ShowtimeListQuery extends PageQuery {
  movie_id?: string;
  hall_id?: string;
  status?: ShowtimeStatus;
  date?: string;
  from?: string;
  to?: string;
  sort?: 'start_at' | 'created_at';
  order?: 'asc' | 'desc';
}

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
  from_price?: number;
}

export interface ShowtimeCancelResult {
  showtime_id: string;
  status: ShowtimeStatus;
  bookings_affected: number;
}
