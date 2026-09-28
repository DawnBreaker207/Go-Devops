import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  PagedData,
  Showtime,
  ShowtimeListItem,
  ShowtimeListQuery,
  ShowtimeCancelResult,
  ShowtimePayload,
} from '@/types';

export const showtimeApi = {
  forMovie: (movieId: string, date?: string) =>
    apiClient
      .get<ApiResponse<ShowtimeListItem[]>>(`/movies/${movieId}/showtimes`, {
        params: date ? { date } : undefined,
      })
      .then(unwrap),

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

  remove: (id: string) =>
    apiClient.delete<ApiResponse<void>>(`/admin/showtimes/${id}`).then(() => undefined),

  cancel: (id: string) =>
    apiClient.post<ApiResponse<ShowtimeCancelResult>>(`/admin/showtimes/${id}/cancel`).then(unwrap),
};
