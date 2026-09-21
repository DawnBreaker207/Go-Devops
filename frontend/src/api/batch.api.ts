import { apiClient, unwrap } from './client';
import type { ApiResponse, BatchJob, BatchJobListQuery, BatchRunResult, PagedData } from '@/types';

export const batchApi = {
  /** Admin only; search filters by job_name. */
  list: (query: BatchJobListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<BatchJob>>>('/admin/batch/jobs', { params: query })
      .then(unwrap),

  /** Enqueues the job (202) without waiting; poll the job list for the result. */
  run: (name: string) =>
    apiClient.post<ApiResponse<BatchRunResult>>(`/admin/batch/jobs/${name}/run`).then(unwrap),
};
