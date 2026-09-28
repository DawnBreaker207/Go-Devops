import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { showtimeApi } from '@/api/showtime.api';
import { movieApi } from '@/api/movie.api';
import { MOVIE_QUERY_KEY } from '@/features/movie/hooks/useMovies';
import { BROWSE_QUERY_KEY } from '@/features/browse/hooks/useBrowse';
import type { ShowtimeListQuery, ShowtimePayload } from '@/types';

export const SHOWTIME_QUERY_KEY = 'showtimes';

const invalidateShowtimeCaches = (queryClient: ReturnType<typeof useQueryClient>) => {
  queryClient.invalidateQueries({ queryKey: [SHOWTIME_QUERY_KEY] });
  queryClient.invalidateQueries({ queryKey: [BROWSE_QUERY_KEY] });
};

// The hall list belongs to the hall domain; re-export for exactly one 'halls' cache key.
export { useHallOptions } from '@/features/hall/hooks/useHalls';

export const useShowtimeList = (query: ShowtimeListQuery) =>
  useQuery({
    queryKey: [SHOWTIME_QUERY_KEY, query],
    queryFn: () => showtimeApi.list(query),
    placeholderData: (previous) => previous,
    // Same admin just edited this list, so skip refetching within one short working session.
    staleTime: 30_000,
  });

/** Shared picker page size (backend hard ceiling is 100). */
export const PICKER_PAGE_SIZE = 100;

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

// Cancelling refunds: the cascade flips paid bookings to refunded, so invalidate the order/booking caches too.
export const useCancelShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => showtimeApi.cancel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [SHOWTIME_QUERY_KEY] });
      void queryClient.invalidateQueries({ queryKey: ['bookings'] });
      void queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
};

export const useDeleteShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => showtimeApi.remove(id),
    onSuccess: () => invalidateShowtimeCaches(queryClient),
  });
};
