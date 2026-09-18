/**
 * GET /api/v1/admin/stats -> dto.AdminStatsResponse. CHI admin (staff nhan 403).
 *
 * Y nghia tung so, theo dung query o backend:
 * - movies / showtimes / users: dem ban ghi chua bi soft-delete. Tai khoan bi
 *   khoa (active=false) VAN duoc dem, giong het meta.total cua GET /admin/users.
 * - bookings: chi dem don `confirmed`. Bang bookings dong thoi la bang giu cho,
 *   nen dem tat ca se toan la hold khong ai tra tien.
 *
 * Day KHONG phai GET /admin/overview: overview tra doanh thu/lap day cua hom nay
 * va canh bao van hanh, con day la kich thuoc catalogue. Hai so "showtimes" cua
 * hai endpoint khong bang nhau va khong nen so sanh.
 */
export interface AdminStats {
  movies: number;
  showtimes: number;
  bookings: number;
  users: number;
}

/**
 * GET /api/v1/admin/reports/daily -> dto.DailyReportResponse. CHI admin.
 *
 * Doc ky truoc khi dung, vi ba dieu duoi day quyet dinh ca man hinh:
 *
 * 1. `days` KHONG phai la moi ngay trong khoang. No la nhung ngay ma job
 *    `closeDay` da chot so (bang `daily_aggregates`), nen mot ngay chua chot
 *    thi VANG MAT hoan toan. "Vang mat" khac han "co dong nhung toan so 0":
 *    ngay khong ban duoc gi van duoc chot va van co dong day du so 0.
 *    Bang rong nghia la job chua chay, khong phai rap khong ban duoc ve nao.
 *
 * 2. Doanh thu va do lap day dem theo HAI TRUC KHAC NHAU. `total_revenue` va
 *    `tickets_sold` dem don `confirmed` co `paid_at` roi vao ngay do; con
 *    `seats_sold` / `capacity` dem cac SUAT CHIEU bat dau trong ngay do. Mot ve
 *    mua hom nay cho suat ngay mai lam tang doanh thu hom nay va do lap day
 *    ngay mai. Hai con so nay khong bao gio phai khop nhau.
 *
 * 3. `occupancy_rate` DA la phan tram (backend lam `ROUND(100.0 * ...)`), khong
 *    phai ty le 0..1. Nhan them 100 se ra 333% thay vi 3.33%.
 */
export interface DailyReport {
  /** YYYY-MM-DD theo gio rap, KHONG phai RFC3339. */
  from: string;
  to: string;
  /** int64 VND nguyen. Tong cua ca khoang, khong phai cua rieng `days`. */
  total_revenue: number;
  tickets_sold: number;
  days: DailyAggregate[];
}

export interface DailyAggregate {
  report_date: string;
  total_revenue: number;
  tickets_sold: number;
  seats_sold: number;
  capacity: number;
  /** Da la phan tram - xem ghi chu 3 o tren. */
  occupancy_rate: number;
  breakdown: DailyBreakdown;
  /** Lan cuoi job chot so ngay nay. RFC3339. */
  updated_at: string;
}

/**
 * Cot `breakdown` la jsonb tu do, khong phai mot struct co rang buoc. Backend
 * hien ghi dung mot khoa `showtimes` (COALESCE ve `[]` nen luon co mat), nhung
 * khong co gi trong schema ep no phai nhu vay mai mai - vi the field nay duoc
 * khai bao tuy chon va moi cho doc deu phai chiu duoc truong hop thieu.
 */
export interface DailyBreakdown {
  showtimes?: DailyBreakdownShowtime[];
}

export interface DailyBreakdownShowtime {
  showtime_id: string;
  /** Ten phim va ten phong duoc CHUP lai luc chot so, nen doi ten phim sau do
   *  khong lam bao cao cu doi theo - va do la dung. */
  movie: string;
  hall: string;
  start_at: string;
  capacity: number;
  seats_sold: number;
  checked_in: number;
  revenue: number;
}

/** Tham so cua GET /admin/reports/daily. Bo trong ca hai = 7 ngay gan nhat. */
export interface DailyReportQuery {
  /** YYYY-MM-DD. Mac dinh la `to` - 6 ngay. */
  from?: string;
  /** YYYY-MM-DD. Mac dinh la hom nay theo gio rap. */
  to?: string;
}
