import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  Combo,
  ComboOrder,
  CreateComboOrderPayload,
  PagedData,
  PageQuery,
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
};
