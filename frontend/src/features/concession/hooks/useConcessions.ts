import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { comboApi } from '@/api/combo.api';
import type { AdminComboListQuery, CreateComboPayload, UpdateComboPayload } from '@/types';

export const CONCESSION_QUERY_KEY = 'concessions';

export const useConcessionList = (query: AdminComboListQuery) =>
  useQuery({
    queryKey: [CONCESSION_QUERY_KEY, query],
    queryFn: () => comboApi.adminList(query),
    placeholderData: (previous) => previous,
  });

/** Every write invalidates the same key; the customer-facing GET /combos is a
 *  separate, uncached public call, so nothing else needs touching here. */
const useInvalidate = () => {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: [CONCESSION_QUERY_KEY] });
};

export const useCreateConcession = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (payload: CreateComboPayload) => comboApi.adminCreate(payload),
    onSuccess: () => void invalidate(),
  });
};

export const useUpdateConcession = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateComboPayload }) =>
      comboApi.adminUpdate(id, payload),
    onSuccess: () => void invalidate(),
  });
};

export const useDeleteConcession = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (id: string) => comboApi.adminDelete(id),
    onSuccess: () => void invalidate(),
  });
};
