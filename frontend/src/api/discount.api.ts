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
  /* --- Customer, on a PENDING order. Requires the customer role. --- */

  /** Every refusal is 400/40001 with its own sentence; show the message, do not
   *  branch on the code. An unknown code and a disabled one answer identically
   *  by design, so this cannot be used to discover which codes exist. */
  apply: (bookingId: string, code: string) =>
    apiClient
      .post<ApiResponse<DiscountApplied>>(`/orders/${bookingId}/discount`, { code })
      .then(unwrap),

  /** Gives the code's redemption back, so a limited code is not burned. */
  remove: (bookingId: string) =>
    apiClient.delete<ApiResponse<DiscountApplied>>(`/orders/${bookingId}/discount`).then(unwrap),

  /* --- Operator catalogue. ADMIN-ONLY, unlike /admin/concessions. --- */

  adminList: (query: DiscountListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<DiscountCode>>>('/admin/discounts', { params: query })
      .then(unwrap),

  adminGet: (id: string) =>
    apiClient.get<ApiResponse<DiscountCode>>(`/admin/discounts/${id}`).then(unwrap),

  adminCreate: (payload: CreateDiscountPayload) =>
    apiClient.post<ApiResponse<DiscountCode>>('/admin/discounts', payload).then(unwrap),

  /** PATCH, not PUT: sends only what changed. An empty body is 400/40001. */
  adminUpdate: (id: string, payload: UpdateDiscountPayload) =>
    apiClient.patch<ApiResponse<DiscountCode>>(`/admin/discounts/${id}`, payload).then(unwrap),

  /** 204 with an EMPTY body - never unwrap this one. */
  adminDelete: (id: string) =>
    apiClient.delete<void>(`/admin/discounts/${id}`).then(() => undefined),
};
