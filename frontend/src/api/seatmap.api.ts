import { apiClient, unwrap } from './client';
import type { ApiResponse, SeatMap } from '@/types';

/** Shared cache key for every seat-map reader. */
export const SEATMAP_QUERY_KEY = 'seatmap';

export const seatMapApi = {
  /** :id is the SHOWTIME id, not the hall. Any authenticated role. Single object, unpaginated. 404/40400 has two meanings distinguished by message only: "showtime not found" vs "showtime is not open". */
  forShowtime: (showtimeId: string) =>
    apiClient.get<ApiResponse<SeatMap>>(`/shows/${showtimeId}/seats`).then(unwrap),
};
