import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { hallApi } from '@/api/hall.api';
import type {
  CloneHallPayload,
  Hall,
  HallPayload,
  HallPrice,
  PageQuery,
  PricePayload,
  UpdateHallPayload,
} from '@/types';
import { SEAT_TYPES } from '@/types';

export const HALL_QUERY_KEY = 'halls';

/** Picker list for form selects. 100 is the backend's hard ceiling (max=100 in
 *  binding; over it is a 400, never truncated). */
export const PICKER_PAGE_SIZE = 100;

export const useHallList = (query: PageQuery) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, query],
    queryFn: () => hallApi.list(query),
    placeholderData: (previous) => previous,
    // The same admin is editing this list (mutations already invalidate proactively) -
    // no refetch-on-mount needed within one short working session.
    staleTime: 30_000,
  });

/** For the hall picker in the showtime form. */
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
    // Templates are in-memory backend constants, no DB touch, never change.
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

export const useHallPrices = (id: string | undefined) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, 'prices', id],
    queryFn: () => hallApi.prices(id as string),
    enabled: Boolean(id),
    staleTime: 5 * 60_000,
  });

/** Per-hall price status used to flag incomplete price sets. */
export const useHallPriceStatuses = (halls: Hall[]) =>
  useQueries({
    queries: halls.map((hall) => ({
      queryKey: [HALL_QUERY_KEY, 'prices', hall.id],
      queryFn: () => hallApi.prices(hall.id),
      staleTime: 5 * 60_000,
    })),
    combine: (results) => {
      const byHallId: Record<string, HallPrice[] | undefined> = {};
      halls.forEach((hall, index) => {
        byHallId[hall.id] = results[index]?.data;
      });
      return { byHallId, isFetching: results.some((r) => r.isFetching) };
    },
  });

/** Price set completeness: all 4 types present, each > 0. */
export const isPriceSetComplete = (prices: HallPrice[] | undefined): boolean =>
  prices !== undefined &&
  SEAT_TYPES.every((type) => {
    const row = prices.find((p) => p.seat_type === type);
    return row !== undefined && row.price > 0;
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

export const useSetHallPrices = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: PricePayload }) =>
      hallApi.setPrices(id, payload),
    onSuccess: (data, variables) => {
      // PUT returns all 4 rows, so write straight into the cache and skip a GET round-trip.
      queryClient.setQueryData([HALL_QUERY_KEY, 'prices', variables.id], data);
      void queryClient.invalidateQueries({ queryKey: [HALL_QUERY_KEY] });
    },
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

/** Seat edits stay in a local draft; the panel saves them in order. */
