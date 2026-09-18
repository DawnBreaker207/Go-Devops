import { apiClient, unwrap } from './client';
import type { AdminStats, ApiResponse, DailyReport, DailyReportQuery } from '@/types';

export const reportApi = {
  /** Chi goi khi role la admin; staff/customer se nhan 403. */
  stats: () => apiClient.get<ApiResponse<AdminStats>>('/admin/stats').then(unwrap),

  /**
   * GET /admin/reports/daily - CHI admin. Tra ve MOT object, khong phan trang
   * va khong phai mang tran.
   *
   * Loi kiem tra tham so la 400/40001 nhung KHONG co `details`: service dung
   * `apperrors.Validation("...")` tran, nen `applyApiFieldErrors` khong gan
   * duoc vao o nhap nao - phai hien bang toast/alert.
   * Bon cau co the gap: "from/to must follow format YYYY-MM-DD",
   * "from must not be after to", "the range can not exceed 366 days".
   */
  daily: (query: DailyReportQuery) =>
    apiClient.get<ApiResponse<DailyReport>>('/admin/reports/daily', { params: query }).then(unwrap),
};
