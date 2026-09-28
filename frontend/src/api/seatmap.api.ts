import { apiClient, unwrap } from './client';
import type { ApiResponse, SeatMap } from '@/types';

export const SEATMAP_QUERY_KEY = 'seatmap';

export const seatMapApi = {
  forShowtime: (showtimeId: string) =>
    apiClient.get<ApiResponse<SeatMap>>(`/shows/${showtimeId}/seats`).then(unwrap),
};
