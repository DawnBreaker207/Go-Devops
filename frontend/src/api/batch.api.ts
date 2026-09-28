import { apiClient, unwrap } from './client';
import type { ApiResponse, BatchJob, BatchJobListQuery, BatchRunResult, PagedData } from '@/types';

export const batchApi = {
  list: (query: BatchJobListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<BatchJob>>>('/admin/batch/jobs', { params: query })
      .then(unwrap),

  run: (name: string) =>
    apiClient.post<ApiResponse<BatchRunResult>>(`/admin/batch/jobs/${name}/run`).then(unwrap),
};
