import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  HoldPayload,
  HoldResult,
  InitResult,
  OrderDetail,
  OrderStatus,
  PagedData,
  PageQuery,
  PayPayload,
  PayResult,
  PaymentProvider,
  RefreshResult,
} from '@/types';

export const orderApi = {
  init: (showId: string) =>
    apiClient.post<ApiResponse<InitResult>>('/orders/init', { show_id: showId }).then(unwrap),

  refresh: (bookingId: string) =>
    apiClient.post<ApiResponse<RefreshResult>>(`/orders/${bookingId}/refresh`).then(unwrap),
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<OrderStatus>>>('/orders', { params: query }).then(unwrap),

  hold: (payload: HoldPayload) =>
    apiClient.post<ApiResponse<HoldResult>>('/orders/hold', payload).then(unwrap),

  pay: (bookingId: string, payload: PayPayload) =>
    apiClient.post<ApiResponse<PayResult>>(`/orders/${bookingId}/pay`, payload).then(unwrap),

  confirm: (bookingId: string) =>
    apiClient.post<ApiResponse<OrderDetail>>(`/orders/${bookingId}/confirm`).then(unwrap),

  cancel: (bookingId: string) =>
    apiClient.post<ApiResponse<OrderStatus>>(`/orders/${bookingId}/cancel`).then(unwrap),

  status: (bookingId: string) =>
    apiClient.get<ApiResponse<OrderStatus>>(`/orders/${bookingId}/status`).then(unwrap),

  detail: (bookingId: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/orders/${bookingId}`).then(unwrap),

  providers: () =>
    apiClient.get<ApiResponse<PaymentProvider[]>>('/payments/providers').then(unwrap),
};
