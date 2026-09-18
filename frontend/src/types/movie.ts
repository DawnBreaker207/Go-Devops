export type MovieStatus = 'draft' | 'showing' | 'ended';

/** Phan loai do tuoi cua Viet Nam. Backend mac dinh 'P' khi gui rong. */
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
  trailer_url: string;
  cast: string;
  age_rating: MovieAgeRating;
  release_date: string;
  status: MovieStatus;
  created_at: string;
  updated_at: string;
}

/**
 * PUT /movies/:id la FULL REPLACE: service gan tat ca cac field tu request, va
 * age_rating rong se bi doi ve 'P'. Form nao sua phim ma bo qua trailer_url /
 * cast / age_rating se xoa sach ba field do. Luon gui du.
 */
export interface MoviePayload {
  title: string;
  genre: string;
  duration: number;
  director: string;
  description?: string;
  poster_url?: string;
  trailer_url?: string;
  cast?: string;
  age_rating?: MovieAgeRating;
  release_date: string;
  status: MovieStatus;
}
