import { useQuery } from '@tanstack/react-query';
import { reportApi } from '@/api/report.api';
import { staffApi } from '@/api/staff.api';
import type { BreakdownQuery, DailyReportQuery } from '@/types';

export const DASHBOARD_QUERY_KEY = 'dashboard';

/** GET /admin/reports/breakdown: daily line, top movies/halls, provider split. Admin only. */
export const useBreakdown = (query: BreakdownQuery, enabled: boolean) =>
  useQuery({
    queryKey: [DASHBOARD_QUERY_KEY, 'breakdown', query.from, query.to],
    queryFn: () => reportApi.breakdown(query),
    enabled,
    refetchOnWindowFocus: false,
    // Keep the previous range on screen while the new one loads (no skeleton flash on filter change).
    placeholderData: (previous) => previous,
  });

/** GET /admin/reports/daily: closed rows with capacity (for occupancy). Admin only. */
export const useDailyReport = (query: DailyReportQuery, enabled: boolean) =>
  useQuery({
    queryKey: [DASHBOARD_QUERY_KEY, 'daily', query.from, query.to],
    queryFn: () => reportApi.daily(query),
    enabled,
    refetchOnWindowFocus: false,
    placeholderData: (previous) => previous,
  });

/** GET /admin/overview: today + 7 days + upcoming showtimes + alerts. Admin only. */
export const useAdminOverview = (enabled: boolean) =>
  useQuery({
    queryKey: [DASHBOARD_QUERY_KEY, 'overview'],
    queryFn: reportApi.overview,
    enabled,
    refetchOnWindowFocus: false,
    placeholderData: (previous) => previous,
  });

/** GET /staff/overview: today's showtimes + counter + pending check-ins. Staff and admin. */
export const useStaffOverview = (enabled: boolean) =>
  useQuery({
    queryKey: [DASHBOARD_QUERY_KEY, 'staff-overview'],
    queryFn: () => staffApi.overview(),
    enabled,
    refetchOnWindowFocus: false,
  });
