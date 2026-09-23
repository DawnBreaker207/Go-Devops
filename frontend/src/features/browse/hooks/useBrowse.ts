import { useQueries, useQuery } from '@tanstack/react-query';
import { movieApi } from '@/api/movie.api';
import { showtimeApi } from '@/api/showtime.api';
import type { PageQuery } from '@/types';

export const BROWSE_QUERY_KEY = 'browse';

/** Public movie list for the home page, filtered client-side. */
export const useNowShowing = (query: PageQuery) =>
  useQuery({
    queryKey: [BROWSE_QUERY_KEY, 'movies', query],
    queryFn: () => movieApi.list(query),
    placeholderData: (previous) => previous,
    staleTime: 60_000,
  });

export const useMovieDetail = (id: string | undefined) =>
  useQuery({
    queryKey: [BROWSE_QUERY_KEY, 'movie', id],
    queryFn: () => movieApi.detail(id as string),
    enabled: Boolean(id),
    staleTime: 60_000,
  });

/** Showtimes of one movie for a single day. */
export const useMovieShowtimes = (id: string | undefined, date?: string) =>
  useQuery({
    queryKey: [BROWSE_QUERY_KEY, 'showtimes', id, date],
    queryFn: () => showtimeApi.forMovie(id as string, date),
    enabled: Boolean(id),
    staleTime: 30_000,
  });

/** Showtimes of one movie across several days, grouped by day. */
export const useMovieShowtimesRange = (id: string | undefined, dates: string[]) => {
  const results = useQueries({
    queries: dates.map((date) => ({
      queryKey: [BROWSE_QUERY_KEY, 'showtimes', id, date],
      queryFn: () => showtimeApi.forMovie(id as string, date),
      enabled: Boolean(id),
      staleTime: 30_000,
    })),
  });

  const isFetching = results.some((result) => result.isFetching);
  const error = results.find((result) => result.error)?.error;
  const groups = dates
    .map((date, i) => ({ date, items: results[i]?.data ?? [] }))
    .filter((group) => group.items.length > 0);

  return { groups, isFetching, error };
};

/** Title search running only after the first character. */
export const useMovieSearch = (search: string) =>
  useQuery({
    queryKey: [BROWSE_QUERY_KEY, 'search', search],
    queryFn: () => movieApi.list({ page: 1, page_size: 20, search }),
    enabled: search.trim().length > 0,
    staleTime: 10_000,
  });
