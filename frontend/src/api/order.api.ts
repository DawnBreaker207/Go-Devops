import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  HoldPayload,
  HoldResult,
  OrderDetail,
  OrderStatus,
  PagedData,
  PageQuery,
  PayPayload,
  PayResult,
  PaymentProvider,
} from '@/types';

export const orderApi = {
  /** GET /orders - KHONG co RequireRoles, tra don cua chinh nguoi goi. CO phan trang. */
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<OrderStatus>>>('/orders', { params: query }).then(unwrap),

  /** POST /orders/hold - RequireRoles(customer) + rate limit rieng (Hold). */
  hold: (payload: HoldPayload) =>
    apiClient.post<ApiResponse<HoldResult>>('/orders/hold', payload).then(unwrap),

  /** POST /orders/:id/pay - tra ve redirect_url cua cong thanh toan. */
  pay: (bookingId: string, payload: PayPayload) =>
    apiClient.post<ApiResponse<PayResult>>(`/orders/${bookingId}/pay`, payload).then(unwrap),

  /** POST /orders/:id/confirm - chot don sau khi cong bao da tra tien.
   *  Chua tra tien thi loi "booking is not paid"; da het han thi "expired". */
  confirm: (bookingId: string) =>
    apiClient.post<ApiResponse<OrderDetail>>(`/orders/${bookingId}/confirm`).then(unwrap),

  cancel: (bookingId: string) =>
    apiClient.post<ApiResponse<OrderStatus>>(`/orders/${bookingId}/cancel`).then(unwrap),

  status: (bookingId: string) =>
    apiClient.get<ApiResponse<OrderStatus>>(`/orders/${bookingId}/status`).then(unwrap),

  /** GET /orders/:id - tra ve CA don VA ve. `GET /orders/:id/tickets` duoc noi
   *  vao CUNG handler nay, nen dung cai nay va doc `.tickets`. */
  detail: (bookingId: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/orders/${bookingId}`).then(unwrap),

  /** GET /payments/providers - JWT thuan, MANG TRAN. */
  providers: () =>
    apiClient.get<ApiResponse<PaymentProvider[]>>('/payments/providers').then(unwrap),
};
