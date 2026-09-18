import type { MovieAgeRating } from './movie';
import type { PageQuery } from './api';

export type ShowtimeStatus = 'open' | 'closed';

export const SHOWTIME_STATUSES: ShowtimeStatus[] = ['open', 'closed'];

/**
 * dto.ShowtimeResponse. Khong field nao omitempty nen KEY luon co mat - nhung
 * gia tri thi khong phai luc nao cung co:
 *
 * - GET /admin/showtimes va GET /admin/showtimes/:id: day du, thoi gian +07:00.
 * - POST va PUT: movie_title / age_rating / hall_name la CHUOI RONG va
 *   created_at / updated_at la zero time '0001-01-01T00:00:00Z'.
 *
 * Vi vay khong bao gio nhet ket qua cua POST/PUT thang vao bang - phai
 * invalidate roi doc lai, neu khong dong vua sua se mat ten phim va ten phong.
 */
export interface Showtime {
  id: string;
  movie_id: string;
  movie_title: string;
  age_rating: MovieAgeRating | '';
  hall_id: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  status: ShowtimeStatus;
  created_at: string;
  updated_at: string;
}

/**
 * Cung mot body cho POST va PUT. movie_id / hall_id / start_at deu `required`
 * o CA HAI verb: doi moi trang thai cung phai gui lai du ba gia tri hien tai.
 * Khong co end_at - backend tu suy ra tu duration cua phim.
 * status: POST nhan roi BO QUA (luon tao 'open'); PUT bo trong thi giu nguyen.
 */
export interface ShowtimePayload {
  movie_id: string;
  hall_id: string;
  start_at: string;
  status?: ShowtimeStatus;
}

export interface ShowtimeListQuery extends PageQuery {
  movie_id?: string;
  hall_id?: string;
  status?: ShowtimeStatus;
  /** YYYY-MM-DD. Uu tien hon from/to, dung gui kem. */
  date?: string;
  from?: string;
  /** YYYY-MM-DD, tinh ca ngay nay. */
  to?: string;
  sort?: 'start_at' | 'created_at';
  order?: 'asc' | 'desc';
}

/**
 * dto.ShowtimeListItem - GET /movies/:id/showtimes va GET /showtimes.
 * CONG KHAI, tra ve MANG TRAN (khong `{items, meta}`).
 *
 * Day la o CHON SUAT cua khach, khac han danh sach van hanh. Ba dieu phai nho:
 *
 * 1. Ket qua bi KEP VAO DUNG MOT NGAY. Khong truyen `date` thi la HOM NAY theo
 *    gio rap, khong phai "tat ca cac suat sap toi". Muon ngay khac phai truyen
 *    `date=YYYY-MM-DD`.
 * 2. Repository loc ngam: phim phai `showing`, suat phai `open`,
 *    `start_at >= now()`, VA phong phai co DU 4 loai gia. Suat vang mat o day
 *    khong co nghia la no khong ton tai.
 * 3. Phim khong o trang thai `showing` tra ve MANG RONG chu khong phai 404.
 *
 * Khac `Showtime` (dto.ShowtimeResponse): co `from_price`, khong co
 * created_at/updated_at.
 */
export interface ShowtimeListItem {
  id: string;
  movie_id: string;
  movie_title: string;
  age_rating: string;
  hall_id: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  status: ShowtimeStatus;
  /** omitempty: VANG MAT khi bang 0, tuc phong chua co gia nao. */
  from_price?: number;
}
