import { apiClient, unwrap } from './client';
import type { ApiResponse, AuditLog, AuditLogListQuery, PagedData } from '@/types';

export const auditApi = {
  /** Admin only. */
  list: (query: AuditLogListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<AuditLog>>>('/admin/audit-logs', { params: query })
      .then(unwrap),
};
