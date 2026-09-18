import { useQuery } from '@tanstack/react-query';
import { reportApi } from '@/api/report.api';

export const DASHBOARD_QUERY_KEY = 'dashboard';

/**
 * GET /admin/stats chi danh cho admin, nen truyen enabled=false khi role khac de
 * khong ban mot request chac chan 403 moi lan mo dashboard.
 */
export const useAdminStats = (enabled: boolean) =>
  useQuery({
    queryKey: [DASHBOARD_QUERY_KEY, 'stats'],
    queryFn: reportApi.stats,
    enabled,
  });
