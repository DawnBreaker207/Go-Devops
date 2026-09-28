export type MovieStatus = 'draft' | 'coming_soon' | 'showing' | 'ended';

export type MovieAgeRating = 'P' | 'K' | 'T13' | 'T16' | 'T18';

export const MOVIE_AGE_RATINGS: MovieAgeRating[] = ['P', 'K', 'T13', 'T16', 'T18'];

export interface Movie {
  id: string;
  title: string;
  genre: string;
  duration: number;
  director: string;
  description: string;
  poster_url: string;
  backdrop_url: string;
  trailer_url: string;
  cast: string;
  age_rating: MovieAgeRating;
  release_date: string;
  status: MovieStatus;
  created_at: string;
  updated_at: string;
}

export interface MoviePayload {
  title: string;
  genre: string;
  duration: number;
  director: string;
  description?: string;
  poster_url?: string;
  backdrop_url?: string;
  trailer_url?: string;
  cast?: string;
  age_rating?: MovieAgeRating;
  release_date: string;
  status: MovieStatus;
}

export interface UploadResult {
  url: string;
  content_type: string;
  size: number;
}

export const POSTER_MAX_BYTES = 5 * 1024 * 1024;
export const POSTER_ACCEPT = ['image/jpeg', 'image/png', 'image/webp'] as const;
