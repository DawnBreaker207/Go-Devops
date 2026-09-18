import { apiClient, unwrap } from './client';
import type { AdminStats, ApiResponse } from '@/types';

export const reportApi = {
  /** Chi goi khi role la admin; staff/customer se nhan 403. */
  stats: () => apiClient.get<ApiResponse<AdminStats>>('/admin/stats').then(unwrap),
};
