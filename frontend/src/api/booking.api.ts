import { apiClient, unwrap } from './client';
import type { AdminOrder, AdminOrderListQuery, ApiResponse, OrderDetail, PagedData } from '@/types';

export const bookingApi = {
  /** GET /admin/orders - CHI admin, staff nhan 403. */
  adminList: (query: AdminOrderListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<AdminOrder>>>('/admin/orders', { params: query })
      .then(unwrap),

  /**
   * GET /staff/orders/:id - duong doc mot don, admin cung goi duoc.
   * Khong tra customer / sold_via / seats: mang tu dong trong bang sang.
   */
  detail: (id: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/staff/orders/${id}`).then(unwrap),
};
