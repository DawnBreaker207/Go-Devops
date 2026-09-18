import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { showtimeApi } from '@/api/showtime.api';
import { movieApi } from '@/api/movie.api';
import type { ShowtimeListQuery, ShowtimePayload } from '@/types';

export const SHOWTIME_QUERY_KEY = 'showtimes';

// Danh sach phong thuoc domain hall; import lai de chi co MOT khoa cache 'halls'
// va mot cho dinh nghia no.
export { useHallOptions } from '@/features/hall/hooks/useHalls';

export const useShowtimeList = (query: ShowtimeListQuery) =>
  useQuery({
    queryKey: [SHOWTIME_QUERY_KEY, query],
    queryFn: () => showtimeApi.list(query),
    placeholderData: (previous) => previous,
  });

/**
 * Danh sach cho o chon trong form. page_size 100 la tran cua backend; du cho
 * quy mo hien tai, va neu vuot thi phai doi sang Select co tim kiem tu xa.
 */
const PICKER_PAGE_SIZE = 100;

export const useMovieOptions = () =>
  useQuery({
    queryKey: ['movies', { page_size: PICKER_PAGE_SIZE }],
    queryFn: () => movieApi.list({ page: 1, page_size: PICKER_PAGE_SIZE }),
    staleTime: 5 * 60_000,
  });

export const useCreateShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ShowtimePayload) => showtimeApi.create(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [SHOWTIME_QUERY_KEY] }),
  });
};

export const useUpdateShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: ShowtimePayload }) =>
      showtimeApi.update(id, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [SHOWTIME_QUERY_KEY] }),
  });
};

export const useDeleteShowtime = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => showtimeApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [SHOWTIME_QUERY_KEY] }),
  });
};
