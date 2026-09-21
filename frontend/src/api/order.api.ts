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
  /** Opens an empty PENDING order (countdown starts at entry). Idempotent while a valid order exists: returns it with `reused`. */
  init: (showId: string) =>
    apiClient.post<ApiResponse<InitResult>>('/orders/init', { show_id: showId }).then(unwrap),

  /** Heartbeat extending the hold within the entry lifetime. */
  refresh: (bookingId: string) =>
    apiClient.post<ApiResponse<RefreshResult>>(`/orders/${bookingId}/refresh`).then(unwrap),
  /** Caller-scoped, no role gate; paginated. Returns OrderStatus WITHOUT tickets; use detail for tickets. */
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<OrderStatus>>>('/orders', { params: query }).then(unwrap),

  /** Requires customer role + dedicated Hold rate limit. */
  hold: (payload: HoldPayload) =>
    apiClient.post<ApiResponse<HoldResult>>('/orders/hold', payload).then(unwrap),

  /** Returns the gateway redirect_url. */
  pay: (bookingId: string, payload: PayPayload) =>
    apiClient.post<ApiResponse<PayResult>>(`/orders/${bookingId}/pay`, payload).then(unwrap),

  /** Settles after the provider reports paid. Unpaid is "booking is not paid"; lapsed is "expired". */
  confirm: (bookingId: string) =>
    apiClient.post<ApiResponse<OrderDetail>>(`/orders/${bookingId}/confirm`).then(unwrap),

  cancel: (bookingId: string) =>
    apiClient.post<ApiResponse<OrderStatus>>(`/orders/${bookingId}/cancel`).then(unwrap),

  status: (bookingId: string) =>
    apiClient.get<ApiResponse<OrderStatus>>(`/orders/${bookingId}/status`).then(unwrap),

  /** Returns order + tickets. /orders/:id/tickets shares this handler, so use this and read `.tickets`. */
  detail: (bookingId: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/orders/${bookingId}`).then(unwrap),

  /** Plain-JWT bare array. */
  providers: () =>
    apiClient.get<ApiResponse<PaymentProvider[]>>('/payments/providers').then(unwrap),
};
