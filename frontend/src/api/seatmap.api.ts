import { apiClient, unwrap } from './client';
import type { ApiResponse, SeatMap } from '@/types';

export const seatMapApi = {
  /**
   * GET /shows/:id/seats - :id la id SUAT CHIEU, khong phai id phong.
   * JWT thuan (moi role da dang nhap). Tra ve MOT object, khong phan trang.
   *
   * 404/40400 co HAI nghia khac nhau chi phan biet duoc bang message:
   * "showtime not found" va "showtime is not open" (suat da dong / da bat dau /
   * phim khong con chieu).
   */
  forShowtime: (showtimeId: string) =>
    apiClient.get<ApiResponse<SeatMap>>(`/shows/${showtimeId}/seats`).then(unwrap),
};
