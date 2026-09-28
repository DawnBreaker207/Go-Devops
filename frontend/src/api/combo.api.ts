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
  list: () => apiClient.get<ApiResponse<Combo[]>>('/combos').then(unwrap),

  createOrder: (payload: CreateComboOrderPayload) =>
    apiClient.post<ApiResponse<ComboOrder>>('/combo-orders', payload).then(unwrap),

  myOrders: (query: PageQuery) =>
    apiClient
      .get<ApiResponse<PagedData<ComboOrder>>>('/combo-orders/me', { params: query })
      .then(unwrap),

  adminList: (query: AdminComboListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<Combo>>>('/admin/concessions', { params: query })
      .then(unwrap),

  adminGet: (id: string) =>
    apiClient.get<ApiResponse<Combo>>(`/admin/concessions/${id}`).then(unwrap),

  adminCreate: (payload: CreateComboPayload) =>
    apiClient.post<ApiResponse<Combo>>('/admin/concessions', payload).then(unwrap),

  adminUpdate: (id: string, payload: UpdateComboPayload) =>
    apiClient.patch<ApiResponse<Combo>>(`/admin/concessions/${id}`, payload).then(unwrap),

  adminDelete: (id: string) =>
    apiClient.delete<void>(`/admin/concessions/${id}`).then(() => undefined),
};
