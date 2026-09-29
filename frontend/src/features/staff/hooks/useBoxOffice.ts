import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { staffApi } from '@/api/staff.api';
import { seatMapApi, SEATMAP_QUERY_KEY } from '@/api/seatmap.api';
import { bookingApi } from '@/api/booking.api';
import { comboApi } from '@/api/combo.api';
import { showtimeApi } from '@/api/showtime.api';
import { PICKER_PAGE_SIZE } from '@/features/showtime/hooks/useShowtimes';
import type {
  CounterComboOrderPayload,
  CounterSellPayload,
  RedeemPayload,
  ShowtimeStatus,
  StaffTicketStatus,
} from '@/types';

export const STAFF_QUERY_KEY = 'staff';

export const useStaffOverview = (date?: string) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'overview', date],
    queryFn: () => staffApi.overview(date),
    // Seats/tickets turn over fast during a shift, so keep a short TTL.
    staleTime: 15_000,
  });

export const useCounterMovies = (date?: string) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'counter-movies', date],
    queryFn: () => staffApi.counterMovies(date),
    staleTime: 15_000,
  });

export const useShowtimeOptions = (search?: string, status?: ShowtimeStatus) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'showtime-options', search, status],
    queryFn: () =>
      showtimeApi.list({
        page: 1,
        page_size: PICKER_PAGE_SIZE,
        status,
        search,
        sort: 'start_at',
        order: 'desc',
      }),
    placeholderData: (previous) => previous,
  });

export const useCounterSeatMap = (showtimeId: string | null) =>
  useQuery({
    queryKey: [SEATMAP_QUERY_KEY, showtimeId],
    queryFn: () => seatMapApi.forShowtime(showtimeId as string),
    enabled: Boolean(showtimeId),
    staleTime: 5_000,
  });

export const useCounterSell = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CounterSellPayload) => staffApi.counterSell(payload),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: [STAFF_QUERY_KEY] });
      queryClient.invalidateQueries({ queryKey: [SEATMAP_QUERY_KEY, variables.show_id] });
    },
  });
};

export const useStaffOrderLookup = () =>
  useMutation({ mutationFn: (id: string) => staffApi.orderDetail(id) });

export const useStaffOrderByTicket = () =>
  useMutation({ mutationFn: (code: string) => staffApi.orderByTicket(code) });

export const useCollectOrderTickets = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => staffApi.orderCollect(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [STAFF_QUERY_KEY] });
    },
  });
};

export const useStaffCustomerSearch = (search: string | undefined) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'customer-search', search ?? ''],
    queryFn: () => staffApi.customers({ page: 1, page_size: 10, search }),
    enabled: Boolean(search && search.trim().length > 0),
  });

export const useStaffCustomerOrders = (customerId: string | undefined) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'customer-orders', customerId],
    queryFn: () => staffApi.customerOrders(customerId as string, { page: 1, page_size: 20 }),
    enabled: Boolean(customerId),
  });

export const useCounterComboSell = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CounterComboOrderPayload) => comboApi.counterSell(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [STAFF_QUERY_KEY] });
    },
  });
};

export const PICKUP_QUERY_KEY = 'combo-pickups';

export const usePendingPickups = (search?: string) =>
  useQuery({
    queryKey: [PICKUP_QUERY_KEY, search ?? ''],
    queryFn: () => comboApi.pendingPickups(search ? { search } : {}),
    placeholderData: (previous) => previous,
  });

export const useCollectComboOrder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => comboApi.collect(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [PICKUP_QUERY_KEY] });
      queryClient.invalidateQueries({ queryKey: [STAFF_QUERY_KEY] });
    },
  });
};

export const useShowtimeTickets = (showtimeId: string | null, status?: StaffTicketStatus) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'tickets', showtimeId, status],
    queryFn: () => staffApi.tickets(showtimeId as string, status),
    enabled: Boolean(showtimeId),
  });

export const useRedeemTicket = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: RedeemPayload }) =>
      bookingApi.redeem(id, payload),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: [STAFF_QUERY_KEY, 'tickets', variables.payload.showtime_id],
      });
      queryClient.invalidateQueries({ queryKey: [STAFF_QUERY_KEY, 'overview'] });
    },
  });
};
