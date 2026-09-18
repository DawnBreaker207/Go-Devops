import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  PagedData,
  Showtime,
  ShowtimeListItem,
  ShowtimeListQuery,
  ShowtimePayload,
} from '@/types';

export const showtimeApi = {
  /**
   * GET /movies/:id/showtimes - CONG KHAI, tra ve MANG TRAN (khong {items,meta}).
   *
   * Day la o chon suat cua khach. Repository chi tra ve suat cua phim dang
   * `showing`, trang thai `open`, `start_at >= now()`, va phong co DU 4 loai
   * gia. Tham so `date` (YYYY-MM-DD) gioi han ket qua trong DUNG mot ngay.
   */
  forMovie: (movieId: string, date?: string) =>
    apiClient
      .get<ApiResponse<ShowtimeListItem[]>>(`/movies/${movieId}/showtimes`, {
        params: date ? { date } : undefined,
      })
      .then(unwrap),

  /** GET /admin/showtimes - admin + staff. Giu ca suat da dong, da qua, phim nhap. */
  list: (query: ShowtimeListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<Showtime>>>('/admin/showtimes', { params: query })
      .then(unwrap),

  detail: (id: string) =>
    apiClient.get<ApiResponse<Showtime>>(`/admin/showtimes/${id}`).then(unwrap),

  create: (payload: ShowtimePayload) =>
    apiClient.post<ApiResponse<Showtime>>('/admin/showtimes', payload).then(unwrap),

  update: (id: string, payload: ShowtimePayload) =>
    apiClient.put<ApiResponse<Showtime>>(`/admin/showtimes/${id}`, payload).then(unwrap),

  /** Tra 200 kem {code,message:"deleted"} va KHONG co data - dung doc ket qua. */
  remove: (id: string) =>
    apiClient.delete<ApiResponse<void>>(`/admin/showtimes/${id}`).then(() => undefined),
};
