import { apiClient, unwrap } from './client';
import type {
  AdminComboListQuery,
  ApiResponse,
  Combo,
  ComboOrder,
  ComboPickup,
  CounterComboOrderPayload,
  CreateComboOrderPayload,
  CreateComboPayload,
  PagedData,
  PageQuery,
  PickupQuery,
  UpdateComboPayload,
} from '@/types';

export const comboApi = {
  list: () => apiClient.get<ApiResponse<Combo[]>>('/combos').then(unwrap),

  createOrder: (payload: CreateComboOrderPayload) =>
    apiClient.post<ApiResponse<ComboOrder>>('/combo-orders', payload).then(unwrap),

  counterSell: (payload: CounterComboOrderPayload) =>
    apiClient.post<ApiResponse<ComboOrder>>('/staff/combo-orders', payload).then(unwrap),

  pendingPickups: (query: PickupQuery) =>
    apiClient
      .get<ApiResponse<ComboPickup[]>>('/staff/combo-orders/pending', { params: query })
      .then(unwrap),

  collect: (id: string) =>
    apiClient.post<ApiResponse<ComboOrder>>(`/staff/combo-orders/${id}/collect`).then(unwrap),

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
