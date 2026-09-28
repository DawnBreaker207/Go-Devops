import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { hallApi } from '@/api/hall.api';
import type { CloneHallPayload, HallPayload, PageQuery, UpdateHallPayload } from '@/types';

export const HALL_QUERY_KEY = 'halls';

// Picker list; 100 is the backend hard ceiling (over it 400s, never truncates).
export const PICKER_PAGE_SIZE = 100;

export const useHallList = (query: PageQuery) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, query],
    queryFn: () => hallApi.list(query),
    placeholderData: (previous) => previous,
    // Same admin just edited this list, so skip refetching within one short working session.
    staleTime: 30_000,
  });

export const useHallOptions = () =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, { page: 1, page_size: PICKER_PAGE_SIZE }],
    queryFn: () => hallApi.list({ page: 1, page_size: PICKER_PAGE_SIZE }),
    staleTime: 5 * 60_000,
  });

export const useHall = (id: string | undefined) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, 'detail', id],
    queryFn: () => hallApi.detail(id as string),
    enabled: Boolean(id),
  });

export const useHallTemplates = () =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, 'templates'],
    // Templates are backend constants that never change; cache indefinitely.
    queryFn: () => hallApi.templates(),
    staleTime: Infinity,
  });

export const useHallSeats = (id: string | undefined) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, 'seats', id],
    queryFn: () => hallApi.seats(id as string),
    enabled: Boolean(id),
    // Every /api/v1 route sends Cache-Control: no-store, so a ~200-seat grid gets
    // no HTTP caching help - keep it in memory longer than the default 30s.
    staleTime: 5 * 60_000,
  });

export const useCreateHall = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: HallPayload) => hallApi.create(payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [HALL_QUERY_KEY] }),
  });
};

export const useUpdateHall = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateHallPayload }) =>
      hallApi.update(id, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [HALL_QUERY_KEY] }),
  });
};

export const useDeleteHall = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => hallApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [HALL_QUERY_KEY] }),
  });
};

export const useCloneHall = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: CloneHallPayload }) =>
      hallApi.clone(id, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [HALL_QUERY_KEY] }),
  });
};

export const useRegenerateLayout = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: HallPayload }) =>
      hallApi.regenerateLayout(id, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [HALL_QUERY_KEY] }),
  });
};
