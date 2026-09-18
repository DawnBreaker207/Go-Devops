import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { hallApi } from '@/api/hall.api';
import type {
  BulkSeatUpdatePayload,
  CloneHallPayload,
  Hall,
  HallPayload,
  HallPrice,
  PageQuery,
  PricePayload,
  Seat,
  UpdateHallPayload,
} from '@/types';
import { SEAT_TYPES } from '@/types';

export const HALL_QUERY_KEY = 'halls';

/** Danh sach cho o chon trong form. 100 la tran cung cua backend (max=100 tren
 *  binding, vuot la 400 chu khong bi cat). */
export const PICKER_PAGE_SIZE = 100;

export const useHallList = (query: PageQuery) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, query],
    queryFn: () => hallApi.list(query),
    placeholderData: (previous) => previous,
  });

/** Dung cho o chon phong trong form suat chieu. */
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
    // Cac mau la hang so trong bo nho cua backend, khong cham DB, khong doi.
    queryFn: () => hallApi.templates(),
    staleTime: Infinity,
  });

export const useHallSeats = (id: string | undefined) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, 'seats', id],
    queryFn: () => hallApi.seats(id as string),
    enabled: Boolean(id),
    // Moi route /api/v1 deu co Cache-Control: no-store, nen luoi ~200 ghe khong
    // duoc HTTP cache ho - giu trong bo nho lau hon mac dinh 30s.
    staleTime: 5 * 60_000,
  });

export const useHallPrices = (id: string | undefined) =>
  useQuery({
    queryKey: [HALL_QUERY_KEY, 'prices', id],
    queryFn: () => hallApi.prices(id as string),
    enabled: Boolean(id),
    staleTime: 5 * 60_000,
  });

/**
 * Bo gia CHUA DU la loi im lang nguy hiem nhat cua nhom endpoint nay: truy van
 * danh sach suat chieu cho khach co `HAVING count(DISTINCT seat_type) = 4`, nen
 * mot phong thieu du mot loai gia se bien mat khoi moi danh sach phia khach ma
 * khong bao gi ca - trong khi man van hanh van thay suat chieu do binh thuong.
 * Backend khong co field nao noi len dieu nay, nen man danh sach phai tu doc gia
 * tung phong. N+1 that, nhung la N <= page_size va moi request chi 0..4 dong.
 */
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

/** Du bo gia hay khong: du 4 loai va moi loai deu > 0. */
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
      // PUT tra ve day du 4 dong nen ghi thang vao cache duoc, khoi mot vong GET.
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

export const useBulkUpdateSeats = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: BulkSeatUpdatePayload }) =>
      hallApi.bulkUpdateSeats(id, payload),
    onSuccess: (touched, variables) => {
      // Response CHI chua nhung ghe bi cham, nen phai tron theo id chu khong
      // duoc thay ca mang - lam vay se mat sach phan con lai cua luoi.
      queryClient.setQueryData<Seat[]>([HALL_QUERY_KEY, 'seats', variables.id], (current) => {
        if (!current) return current;
        const updated = new Map(touched.map((s) => [s.id, s]));
        return current.map((s) => updated.get(s.id) ?? s);
      });
    },
  });
};
