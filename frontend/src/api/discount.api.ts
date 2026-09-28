import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  CreateDiscountPayload,
  DiscountApplied,
  DiscountCode,
  DiscountListQuery,
  PagedData,
  UpdateDiscountPayload,
} from '@/types';

export const discountApi = {
  apply: (bookingId: string, code: string) =>
    apiClient
      .post<ApiResponse<DiscountApplied>>(`/orders/${bookingId}/discount`, { code })
      .then(unwrap),

  remove: (bookingId: string) =>
    apiClient.delete<ApiResponse<DiscountApplied>>(`/orders/${bookingId}/discount`).then(unwrap),

  adminList: (query: DiscountListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<DiscountCode>>>('/admin/discounts', { params: query })
      .then(unwrap),

  adminGet: (id: string) =>
    apiClient.get<ApiResponse<DiscountCode>>(`/admin/discounts/${id}`).then(unwrap),

  adminCreate: (payload: CreateDiscountPayload) =>
    apiClient.post<ApiResponse<DiscountCode>>('/admin/discounts', payload).then(unwrap),

  adminUpdate: (id: string, payload: UpdateDiscountPayload) =>
    apiClient.patch<ApiResponse<DiscountCode>>(`/admin/discounts/${id}`, payload).then(unwrap),

  adminDelete: (id: string) =>
    apiClient.delete<void>(`/admin/discounts/${id}`).then(() => undefined),
};
