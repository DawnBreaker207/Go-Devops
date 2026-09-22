import { useQuery } from '@tanstack/react-query';
import { reportApi } from '@/api/report.api';
import type { DailyReportQuery } from '@/types';

export const REPORT_QUERY_KEY = 'reports';

/** Daily report kept cached longer since it changes only on close-out. */
export const useDailyReport = (query: DailyReportQuery) =>
  useQuery({
    queryKey: [REPORT_QUERY_KEY, 'daily', query],
    queryFn: () => reportApi.daily(query),
    placeholderData: (previous) => previous,
    staleTime: 5 * 60_000,
  });
