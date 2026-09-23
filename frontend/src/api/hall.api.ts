import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  BulkSeatUpdatePayload,
  CloneHallPayload,
  Hall,
  HallPayload,
  HallPrice,
  HallTemplate,
  MergeSeatsPayload,
  PagedData,
  PageQuery,
  PricePayload,
  PublicPriceList,
  Seat,
  SeatUpdatePayload,
  SplitSeatPayload,
  UpdateHallPayload,
} from '@/types';

export const hallApi = {
  /** PUBLIC price page: no auth, one payload, only bookable halls. */
  publicPrices: () => apiClient.get<ApiResponse<PublicPriceList>>('/pricing').then(unwrap),

  /** Paginated; page_size > 100 is rejected 400/40001, not clamped. */
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<Hall>>>('/admin/halls', { params: query }).then(unwrap),

  detail: (id: string) => apiClient.get<ApiResponse<Hall>>(`/admin/halls/${id}`).then(unwrap),

  create: (payload: HallPayload) =>
    apiClient.post<ApiResponse<Hall>>('/admin/halls', payload).then(unwrap),

  /** Meta only (name/screen/aisles/active); never touches seats. */
  update: (id: string, payload: UpdateHallPayload) =>
    apiClient.put<ApiResponse<Hall>>(`/admin/halls/${id}`, payload).then(unwrap),

  /** 204 with empty body; never unwrap. */
  remove: (id: string) =>
    apiClient
      .delete<void>(`/admin/halls/${id}`, { headers: { Accept: '*/*' } })
      .then(() => undefined),

  clone: (id: string, payload: CloneHallPayload) =>
    apiClient.post<ApiResponse<Hall>>(`/admin/halls/${id}/clone`, payload).then(unwrap),

  /** Destructive: wipes seats + showtime_seats of non-deleted shows, then rebuilds. 409 if the hall ever had any order. */
  regenerateLayout: (id: string, payload: HallPayload) =>
    apiClient.put<ApiResponse<Hall>>(`/admin/halls/${id}/layout`, payload).then(unwrap),

  /** Bare array of 3, alphabetical. */
  templates: () => apiClient.get<ApiResponse<HallTemplate[]>>('/admin/hall-templates').then(unwrap),

  /** Whole grid in one call, unpaginated. */
  seats: (id: string) =>
    apiClient.get<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats`).then(unwrap),

  /** Returns only touched seats, not the whole grid. */
  bulkUpdateSeats: (id: string, payload: BulkSeatUpdatePayload) =>
    apiClient.patch<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats`, payload).then(unwrap),

  updateSeat: (id: string, seatId: string, payload: SeatUpdatePayload) =>
    apiClient.put<ApiResponse<Seat>>(`/admin/halls/${id}/seats/${seatId}`, payload).then(unwrap),

  /** Appends exactly one standard row; returns only the new row. */
  addRow: (id: string) =>
    apiClient.post<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/rows`).then(unwrap),

  /** Deletes any row; rows behind renumber. Returns the deleted seats. */
  deleteRow: (id: string, rowLabel: string) =>
    apiClient.delete<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/rows/${rowLabel}`).then(unwrap),

  /** Merges 2 adjacent singles into a couple; the right seat is deleted. */
  mergeSeats: (id: string, payload: MergeSeatsPayload) =>
    apiClient.post<ApiResponse<Seat>>(`/admin/halls/${id}/seats/merge`, payload).then(unwrap),

  /** Splits a couple into 2 singles. Returns both (old + new in the next column). */
  splitSeat: (id: string, payload: SplitSeatPayload) =>
    apiClient.post<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/split`, payload).then(unwrap),

  /** Bare array of 0..4, alphabetical. */
  prices: (id: string) =>
    apiClient.get<ApiResponse<HallPrice[]>>(`/admin/halls/${id}/prices`).then(unwrap),

  /** Write side is a MAP: all 4 types required, each > 0. Returns 4 rows in AllSeatTypes order (not GET order). */
  setPrices: (id: string, payload: PricePayload) =>
    apiClient.put<ApiResponse<HallPrice[]>>(`/admin/halls/${id}/prices`, payload).then(unwrap),
};
