import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { staffApi } from '@/api/staff.api';
import { seatMapApi, SEATMAP_QUERY_KEY } from '@/api/seatmap.api';
import { bookingApi } from '@/api/booking.api';
import { showtimeApi } from '@/api/showtime.api';
import type { CounterSellPayload, RedeemPayload, ShowtimeStatus, StaffTicketStatus } from '@/types';

export const STAFF_QUERY_KEY = 'staff';

/** GET /staff/overview - one-day dashboard + counter sales + pending check-ins. */
export const useStaffOverview = (date?: string) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'overview', date],
    queryFn: () => staffApi.overview(date),
    // Seats/tickets turn over fast during a shift; keep data fresh for a short window.
    staleTime: 15_000,
  });

/** Showtime picker with optional status filter. */
const SHOWTIME_PICKER_PAGE_SIZE = 100;
export const useShowtimeOptions = (search?: string, status?: ShowtimeStatus) =>
  useQuery({
    queryKey: [STAFF_QUERY_KEY, 'showtime-options', search, status],
    queryFn: () =>
      showtimeApi.list({
        page: 1,
        page_size: SHOWTIME_PICKER_PAGE_SIZE,
        status,
        search,
        sort: 'start_at',
        order: 'desc',
      }),
    placeholderData: (previous) => previous,
  });

/** GET /shows/:id/seats - seat map of the showtime picked in the counter-sale form. */
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

/** GET /staff/showtimes/:id/tickets - ticket list of one showtime, issued/redeemed filter. */
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
