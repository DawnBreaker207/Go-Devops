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
  overview: () => apiClient.get<ApiResponse<AdminOverview>>('/admin/overview').then(unwrap),

  daily: (query: DailyReportQuery) =>
    apiClient.get<ApiResponse<DailyReport>>('/admin/reports/daily', { params: query }).then(unwrap),

  breakdown: (query: BreakdownQuery) =>
    apiClient
      .get<ApiResponse<Breakdown>>('/admin/reports/breakdown', { params: query })
      .then(unwrap),
};
