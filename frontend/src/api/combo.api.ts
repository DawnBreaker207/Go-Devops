import { apiClient, unwrap } from './client';
import type {
  AdminComboListQuery,
  ApiResponse,
  Combo,
  ComboOrder,
  CreateComboOrderPayload,
  CreateComboPayload,
  PagedData,
  PageQuery,
  UpdateComboPayload,
} from '@/types';

export const comboApi = {
  /** Public; returns a bare array. */
  list: () => apiClient.get<ApiResponse<Combo[]>>('/combos').then(unwrap),

  /** Requires customer role. Independent from /orders/*. */
  createOrder: (payload: CreateComboOrderPayload) =>
    apiClient.post<ApiResponse<ComboOrder>>('/combo-orders', payload).then(unwrap),

  /** Requires customer role; paginated. */
  myOrders: (query: PageQuery) =>
    apiClient
      .get<ApiResponse<PagedData<ComboOrder>>>('/combo-orders/me', { params: query })
      .then(unwrap),

  /* --- Operator catalogue. Admin AND staff, same scope as /admin/halls. --- */

  /** Paged AND includes inactive products, which `list` above never returns. */
  adminList: (query: AdminComboListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<Combo>>>('/admin/concessions', { params: query })
      .then(unwrap),

  adminGet: (id: string) =>
    apiClient.get<ApiResponse<Combo>>(`/admin/concessions/${id}`).then(unwrap),

  adminCreate: (payload: CreateComboPayload) =>
    apiClient.post<ApiResponse<Combo>>('/admin/concessions', payload).then(unwrap),

  /** PATCH, not PUT: sends only what changed. An empty body is 400/40001. */
  adminUpdate: (id: string, payload: UpdateComboPayload) =>
    apiClient.patch<ApiResponse<Combo>>(`/admin/concessions/${id}`, payload).then(unwrap),

  /** 204 with an EMPTY body - never unwrap this one. */
  adminDelete: (id: string) =>
    apiClient.delete<void>(`/admin/concessions/${id}`).then(() => undefined),
};
