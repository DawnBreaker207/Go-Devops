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
