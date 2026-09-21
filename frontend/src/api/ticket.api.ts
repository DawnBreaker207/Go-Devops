import { apiClient, unwrap } from './client';
import type { ApiResponse, TicketQR } from '@/types';

export const ticketApi = {
  /** Base64 PNG QR for one ticket. Any authenticated role; works even after showtime. */
  qr: (ticketId: string) =>
    apiClient.get<ApiResponse<TicketQR>>(`/tickets/${ticketId}/qr`).then(unwrap),
};
