import { useQuery } from '@tanstack/react-query';
import { reportApi } from '@/api/report.api';
import type { DailyReportQuery } from '@/types';

export const REPORT_QUERY_KEY = 'reports';

/**
 * GET /admin/reports/daily doc bang `daily_aggregates` - mot bang chi doi khi
 * job `closeDay` chay (23:59 moi ngay, hoac khi bam chay tay). Khong co ich gi
 * khi refetch lien tuc, nen giu lau hon mac dinh 30s.
 */
export const useDailyReport = (query: DailyReportQuery) =>
  useQuery({
    queryKey: [REPORT_QUERY_KEY, 'daily', query],
    queryFn: () => reportApi.daily(query),
    placeholderData: (previous) => previous,
    staleTime: 5 * 60_000,
  });
