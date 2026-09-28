import { apiClient, unwrap } from './client';
import type { ApiResponse, TicketQR } from '@/types';

export const ticketApi = {
  qr: (ticketId: string) =>
    apiClient.get<ApiResponse<TicketQR>>(`/tickets/${ticketId}/qr`).then(unwrap),
};
