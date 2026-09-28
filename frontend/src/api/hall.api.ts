import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  BulkSeatUpdatePayload,
  CloneHallPayload,
  Hall,
  HallPayload,
  HallTemplate,
  MergeSeatsPayload,
  PagedData,
  PageQuery,
  Seat,
  SeatUpdatePayload,
  SplitSeatPayload,
  UpdateHallPayload,
} from '@/types';

export const hallApi = {
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<Hall>>>('/admin/halls', { params: query }).then(unwrap),

  detail: (id: string) => apiClient.get<ApiResponse<Hall>>(`/admin/halls/${id}`).then(unwrap),

  create: (payload: HallPayload) =>
    apiClient.post<ApiResponse<Hall>>('/admin/halls', payload).then(unwrap),

  update: (id: string, payload: UpdateHallPayload) =>
    apiClient.put<ApiResponse<Hall>>(`/admin/halls/${id}`, payload).then(unwrap),

  remove: (id: string) =>
    apiClient
      .delete<void>(`/admin/halls/${id}`, { headers: { Accept: '*/*' } })
      .then(() => undefined),

  clone: (id: string, payload: CloneHallPayload) =>
    apiClient.post<ApiResponse<Hall>>(`/admin/halls/${id}/clone`, payload).then(unwrap),

  regenerateLayout: (id: string, payload: HallPayload) =>
    apiClient.put<ApiResponse<Hall>>(`/admin/halls/${id}/layout`, payload).then(unwrap),

  templates: () => apiClient.get<ApiResponse<HallTemplate[]>>('/admin/hall-templates').then(unwrap),

  seats: (id: string) =>
    apiClient.get<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats`).then(unwrap),

  bulkUpdateSeats: (id: string, payload: BulkSeatUpdatePayload) =>
    apiClient.patch<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats`, payload).then(unwrap),

  updateSeat: (id: string, seatId: string, payload: SeatUpdatePayload) =>
    apiClient.put<ApiResponse<Seat>>(`/admin/halls/${id}/seats/${seatId}`, payload).then(unwrap),

  addRow: (id: string) =>
    apiClient.post<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/rows`).then(unwrap),

  addColumn: (id: string) =>
    apiClient.post<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/columns`).then(unwrap),

  deleteRow: (id: string, rowLabel: string) =>
    apiClient.delete<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/rows/${rowLabel}`).then(unwrap),

  mergeSeats: (id: string, payload: MergeSeatsPayload) =>
    apiClient.post<ApiResponse<Seat>>(`/admin/halls/${id}/seats/merge`, payload).then(unwrap),

  splitSeat: (id: string, payload: SplitSeatPayload) =>
    apiClient.post<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats/split`, payload).then(unwrap),
};
