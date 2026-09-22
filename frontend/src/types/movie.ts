export type MovieStatus = 'draft' | 'coming_soon' | 'showing' | 'ended';

/** Vietnam age ratings. Backend defaults to 'P' on empty. */
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

/** PUT /movies/:id is FULL REPLACE (empty age_rating resets to 'P'); omitted trailer_url/cast/age_rating are wiped. Always send complete. */
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

/** dto.UploadResponse from POST /admin/uploads/poster. `url` is what goes into
 *  `poster_url`; it is served from /media/* which sits OUTSIDE the /api/v1 prefix. */
export interface UploadResult {
  url: string;
  content_type: string;
  size: number;
}

/** Backend limits, mirrored so the form can refuse before spending the upload.
 *  Size is `storage.max_upload_mb` in config.yaml (5 today). */
export const POSTER_MAX_BYTES = 5 * 1024 * 1024;
export const POSTER_ACCEPT = ['image/jpeg', 'image/png', 'image/webp'] as const;
