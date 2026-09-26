import type { Movie } from '@/types';

/** Landscape art for the home hero, the wide cards and the trailer player.
 *
 *  `backdrop_url` arrived with migration `000011`, so it is empty on every movie created before it
 *  and on any movie an operator has not given one. The portrait poster then stands in: it crops in a
 *  3:2 box, but a cropped poster beats an empty frame, and the operator can fix it by uploading a
 *  backdrop. Exported (and tested) rather than inlined because three different surfaces need the
 *  same rule. */
export const movieBackdrop = (movie: Pick<Movie, 'backdrop_url' | 'poster_url'>): string =>
  movie.backdrop_url || movie.poster_url || '';

/** `release_date` is one of the backend's deliberate `YYYY-MM-DD` strings, NOT an RFC3339 instant,
 *  so slicing the year is safe here — and safer than parsing it, since running a date-only string
 *  through a timezone can shift it across a year boundary. */
export const movieYear = (movie: Pick<Movie, 'release_date'>): string =>
  /^\d{4}-\d{2}-\d{2}/.test(movie.release_date ?? '') ? movie.release_date.slice(0, 4) : '';
