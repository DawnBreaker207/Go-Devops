import { apiClient, unwrap } from './client';
import type { ApiResponse, PagedData, Showtime, ShowtimeListQuery, ShowtimePayload } from '@/types';

export const showtimeApi = {
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
