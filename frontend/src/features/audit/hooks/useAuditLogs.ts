import { useQuery } from '@tanstack/react-query';
import { auditApi } from '@/api/audit.api';
import type { AuditLogListQuery } from '@/types';

export const AUDIT_QUERY_KEY = 'audit-logs';

export const useAuditLogList = (query: AuditLogListQuery) =>
  useQuery({
    queryKey: [AUDIT_QUERY_KEY, query],
    queryFn: () => auditApi.list(query),
    placeholderData: (previous) => previous,
  });
