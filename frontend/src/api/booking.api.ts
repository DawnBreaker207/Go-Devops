import { apiClient, unwrap } from './client';
import type {
  AdminOrder,
  AdminOrderListQuery,
  ApiResponse,
  OrderDetail,
  PagedData,
  RedeemPayload,
  RedeemResult,
} from '@/types';

export const bookingApi = {
  /** Admin only; staff gets 403. */
  adminList: (query: AdminOrderListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<AdminOrder>>>('/admin/orders', { params: query })
      .then(unwrap),

  /** Single-order read for staff+admin. Omits customer/sold_via/seats; join from tables. */
  detail: (id: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/staff/orders/${id}`).then(unwrap),

  /** Gate check-in. `id` accepts a ticket id or a scanned QR. Staff+admin. */
  redeem: (id: string, payload: RedeemPayload) =>
    apiClient.post<ApiResponse<RedeemResult>>(`/tickets/${id}/redeem`, payload).then(unwrap),
};
