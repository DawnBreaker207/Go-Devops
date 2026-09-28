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
  adminList: (query: AdminOrderListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<AdminOrder>>>('/admin/orders', { params: query })
      .then(unwrap),

  detail: (id: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/staff/orders/${id}`).then(unwrap),

  redeem: (id: string, payload: RedeemPayload) =>
    apiClient.post<ApiResponse<RedeemResult>>(`/tickets/${id}/redeem`, payload).then(unwrap),
};
