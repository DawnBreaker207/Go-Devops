import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  BoxOfficeDay,
  CounterSellPayload,
  OrderDetail,
  OrderStatus,
  PagedData,
  PageQuery,
  StaffBoard,
  StaffOverview,
  StaffTicket,
  StaffTicketStatus,
  User,
} from '@/types';

export const staffApi = {
  dashboard: (date?: string) =>
    apiClient
      .get<ApiResponse<StaffBoard>>('/staff/dashboard', { params: date ? { date } : undefined })
      .then(unwrap),

  overview: (date?: string) =>
    apiClient
      .get<ApiResponse<StaffOverview>>('/staff/overview', { params: date ? { date } : undefined })
      .then(unwrap),

  boxOfficeDay: (date?: string) =>
    apiClient
      .get<ApiResponse<BoxOfficeDay>>('/staff/boxoffice/day', {
        params: date ? { date } : undefined,
      })
      .then(unwrap),

  // status: issued | redeemed; empty means both.
  tickets: (showtimeId: string, status?: StaffTicketStatus) =>
    apiClient
      .get<ApiResponse<StaffTicket[]>>(`/staff/showtimes/${showtimeId}/tickets`, {
        params: status ? { status } : undefined,
      })
      .then(unwrap),

  // Confirmed immediately, no email sent.
  counterSell: (payload: CounterSellPayload) =>
    apiClient.post<ApiResponse<OrderDetail>>('/staff/orders', payload).then(unwrap),

  // Any order, reconciled with the gateway.
  orderDetail: (id: string) =>
    apiClient.get<ApiResponse<OrderDetail>>(`/staff/orders/${id}`).then(unwrap),

  customers: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<User>>>('/staff/customers', { params: query }).then(unwrap),

  // 404 unless the id is a customer account.
  customerProfile: (id: string) =>
    apiClient.get<ApiResponse<User>>(`/staff/customers/${id}`).then(unwrap),

  customerOrders: (id: string, query: PageQuery) =>
    apiClient
      .get<ApiResponse<PagedData<OrderStatus>>>(`/staff/customers/${id}/orders`, {
        params: query,
      })
      .then(unwrap),
};
