import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { showtimeApi } from '@/api/showtime.api';
import { movieApi } from '@/api/movie.api';
import { MOVIE_QUERY_KEY } from '@/features/movie/hooks/useMovies';
import { BROWSE_QUERY_KEY } from '@/features/browse/hooks/useBrowse';
import type { ShowtimeListQuery, ShowtimePayload } from '@/types';

export const SHOWTIME_QUERY_KEY = 'showtimes';

/** Refresh all showtime caches after a mutation. */
const invalidateShowtimeCaches = (queryClient: ReturnType<typeof useQueryClient>) => {
  queryClient.invalidateQueries({ queryKey: [SHOWTIME_QUERY_KEY] });
  queryClient.invalidateQueries({ queryKey: [BROWSE_QUERY_KEY] });
};

// The hall list belongs to the hall domain; re-export so there is exactly ONE 'halls' cache key
// and one place defining it.
export { useHallOptions } from '@/features/hall/hooks/useHalls';

export const useShowtimeList = (query: ShowtimeListQuery) =>
  useQuery({
    queryKey: [SHOWTIME_QUERY_KEY, query],
    queryFn: () => showtimeApi.list(query),
    placeholderData: (previous) => previous,
    // The same admin is editing this list (proactive invalidate above) -
    // no refetch-on-mount needed within one short working session.
    staleTime: 30_000,
  });

/** Movie picker list for the showtime form. */
const PICKER_PAGE_SIZE = 100;

export const useMovieOptions = () =>
  useQuery({
    queryKey: [MOVIE_QUERY_KEY, { page_size: PICKER_PAGE_SIZE }],
    queryFn: () => movieApi.list({ page: 1, page_size: PICKER_PAGE_SIZE }),
    staleTime: 5 * 60_000,
  });

export const useCreateShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ShowtimePayload) => showtimeApi.create(payload),
    onSuccess: () => invalidateShowtimeCaches(queryClient),
  });
};

export const useUpdateShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: ShowtimePayload }) =>
      showtimeApi.update(id, payload),
    onSuccess: () => invalidateShowtimeCaches(queryClient),
  });
};

export const useDeleteShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => showtimeApi.remove(id),
    onSuccess: () => invalidateShowtimeCaches(queryClient),
  });
};
