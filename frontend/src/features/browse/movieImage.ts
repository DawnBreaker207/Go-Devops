import type { Movie } from '@/types';

// Landscape falls back to poster: pre-000011 movies have empty backdrops.
export const movieBackdrop = (movie: Pick<Movie, 'backdrop_url' | 'poster_url'>): string =>
  movie.backdrop_url || movie.poster_url || '';

// release_date is date-only: slice the year instead of parsing, which shifts timezones.
export const movieYear = (movie: Pick<Movie, 'release_date'>): string =>
  /^\d{4}-\d{2}-\d{2}/.test(movie.release_date ?? '') ? movie.release_date.slice(0, 4) : '';
