import { useQuery } from '@tanstack/react-query';
import { movieApi } from '@/api/movie.api';
import { showtimeApi } from '@/api/showtime.api';
import type { PageQuery } from '@/types';

export const BROWSE_QUERY_KEY = 'browse';

/**
 * GET /movies la CONG KHAI (nhom `public` + OptionalAuth trong router.go), nen
 * trang chu chay duoc khi chua dang nhap. No tra ve MOI phim ke ca `draft` va
 * `ended`, khong loc gi ca - phia khach phai tu loc `showing`.
 */
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

/**
 * GET /movies/:id/showtimes - cong khai, MANG TRAN (khong phan trang).
 *
 * Day la o CHON SUAT cua khach, khong phai danh sach van hanh: repository loc
 * `movies.status='showing'` VA `showtimes.status='open'` VA `start_at >= now()`
 * VA phong phai co DU 4 loai gia. Mot suat bien mat khoi day khong co nghia la
 * no khong ton tai - man /showtimes phia van hanh moi thay het.
 */
export const useMovieShowtimes = (id: string | undefined, date?: string) =>
  useQuery({
    queryKey: [BROWSE_QUERY_KEY, 'showtimes', id, date],
    queryFn: () => showtimeApi.forMovie(id as string, date),
    enabled: Boolean(id),
    staleTime: 30_000,
  });
