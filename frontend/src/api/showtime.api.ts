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
  /** Public bare array for the customer showtime picker. Only `showing` movies, `open` shows with start_at >= now and 4 price rows; `date` (YYYY-MM-DD) limits to one day. */
  forMovie: (movieId: string, date?: string) =>
    apiClient
      .get<ApiResponse<ShowtimeListItem[]>>(`/movies/${movieId}/showtimes`, {
        params: date ? { date } : undefined,
      })
      .then(unwrap),

  /** Admin + staff. Keeps closed, past, and draft-movie shows. */
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

  /** 200 with {code,message:"deleted"} and no data; ignore the body. */
  remove: (id: string) =>
    apiClient.delete<ApiResponse<void>>(`/admin/showtimes/${id}`).then(() => undefined),

  /** The ONLY way to stop a showtime that has already sold tickets.
   *  DELETE refuses with 409 once a showtime has any booking; this cancels it and
   *  refunds every paid booking through the normal refund pipeline, voiding the
   *  issued tickets. It moves real money — never call it without a confirmation. */
  cancel: (id: string) =>
    apiClient.post<ApiResponse<ShowtimeCancelResult>>(`/admin/showtimes/${id}/cancel`).then(unwrap),
};
