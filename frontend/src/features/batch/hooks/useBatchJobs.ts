import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { batchApi } from '@/api/batch.api';
import type { BatchJobListQuery } from '@/types';

export const BATCH_QUERY_KEY = 'batch-jobs';

export const useBatchJobList = (query: BatchJobListQuery) =>
  useQuery({
    queryKey: [BATCH_QUERY_KEY, query],
    queryFn: () => batchApi.list(query),
    placeholderData: (previous) => previous,
    // A job run history can still be 'running'; poll periodically so results
    // appear without the user refreshing manually.
    refetchInterval: 10_000,
  });

export const useRunBatchJob = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => batchApi.run(name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [BATCH_QUERY_KEY] }),
  });
};
