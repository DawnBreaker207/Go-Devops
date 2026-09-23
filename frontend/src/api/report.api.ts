import { apiClient, unwrap } from './client';
import type {
  AdminOverview,
  ApiResponse,
  Breakdown,
  BreakdownQuery,
  DailyReport,
  DailyReportQuery,
} from '@/types';

export const reportApi = {
  /** Today + 7 days + upcoming shows + alerts. Admin only. */
  overview: () => apiClient.get<ApiResponse<AdminOverview>>('/admin/overview').then(unwrap),

  /** Single object, unpaginated. Param errors are bare 400/40001 without `details`, so show toast/alert (applyApiFieldErrors cannot bind them). */
  daily: (query: DailyReportQuery) =>
    apiClient.get<ApiResponse<DailyReport>>('/admin/reports/daily', { params: query }).then(unwrap),

  /** Paid-money analytics: daily line, top movies/halls, provider split. Admin only. */
  breakdown: (query: BreakdownQuery) =>
    apiClient
      .get<ApiResponse<Breakdown>>('/admin/reports/breakdown', { params: query })
      .then(unwrap),
};
