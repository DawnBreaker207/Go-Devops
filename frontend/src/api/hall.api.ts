import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  BulkSeatUpdatePayload,
  CloneHallPayload,
  Hall,
  HallPayload,
  HallPrice,
  HallTemplate,
  PagedData,
  PageQuery,
  PricePayload,
  Seat,
  SeatUpdatePayload,
  UpdateHallPayload,
} from '@/types';

export const hallApi = {
  /** GET /admin/halls - CO phan trang, khong phai mang tran. admin + staff.
   *  page_size > 100 bi tu choi 400/40001 chu KHONG bi cat bot. */
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<Hall>>>('/admin/halls', { params: query }).then(unwrap),

  detail: (id: string) => apiClient.get<ApiResponse<Hall>>(`/admin/halls/${id}`).then(unwrap),

  create: (payload: HallPayload) =>
    apiClient.post<ApiResponse<Hall>>('/admin/halls', payload).then(unwrap),

  /** PUT /admin/halls/:id - chi ten / huong man / loi di / active. Khong dung toi ghe. */
  update: (id: string, payload: UpdateHallPayload) =>
    apiClient.put<ApiResponse<Hall>>(`/admin/halls/${id}`, payload).then(unwrap),

  /** Tra 204 VOI THAN RONG - khong co envelope. Dung cho qua unwrap(). */
  remove: (id: string) =>
    apiClient
      .delete<void>(`/admin/halls/${id}`, { headers: { Accept: '*/*' } })
      .then(() => undefined),

  clone: (id: string, payload: CloneHallPayload) =>
    apiClient.post<ApiResponse<Hall>>(`/admin/halls/${id}/clone`, payload).then(unwrap),

  /**
   * PUT /admin/halls/:id/layout - dung lai TOAN BO luoi ghe. Pha huy: xoa het
   * seats va showtime_seats cua moi suat chieu chua xoa, roi sinh lai. Bi chan
   * vinh vien 409 khi phong TUNG co bat ky don nao (khong loc trang thai,
   * khong loc suat da xoa).
   */
  regenerateLayout: (id: string, payload: HallPayload) =>
    apiClient.put<ApiResponse<Hall>>(`/admin/halls/${id}/layout`, payload).then(unwrap),

  /** GET /admin/hall-templates - MANG TRAN, 3 muc, sap theo alphabet. */
  templates: () => apiClient.get<ApiResponse<HallTemplate[]>>('/admin/hall-templates').then(unwrap),

  /** GET /admin/halls/:id/seats - MANG TRAN, ca luoi trong mot lan, khong phan trang. */
  seats: (id: string) =>
    apiClient.get<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats`).then(unwrap),

  /** PATCH /admin/halls/:id/seats - tra ve CHI nhung ghe bi cham, khong phai ca luoi. */
  bulkUpdateSeats: (id: string, payload: BulkSeatUpdatePayload) =>
    apiClient.patch<ApiResponse<Seat[]>>(`/admin/halls/${id}/seats`, payload).then(unwrap),

  updateSeat: (id: string, seatId: string, payload: SeatUpdatePayload) =>
    apiClient.put<ApiResponse<Seat>>(`/admin/halls/${id}/seats/${seatId}`, payload).then(unwrap),

  /** GET /admin/halls/:id/prices - MANG TRAN 0..4 muc, sap theo ALPHABET. */
  prices: (id: string) =>
    apiClient.get<ApiResponse<HallPrice[]>>(`/admin/halls/${id}/prices`).then(unwrap),

  /** PUT /admin/halls/:id/prices - phia ghi la MAP, phai du 4 loai va deu > 0.
   *  Tra ve mang 4 muc theo thu tu AllSeatTypes, khong phai thu tu cua GET. */
  setPrices: (id: string, payload: PricePayload) =>
    apiClient.put<ApiResponse<HallPrice[]>>(`/admin/halls/${id}/prices`, payload).then(unwrap),
};
