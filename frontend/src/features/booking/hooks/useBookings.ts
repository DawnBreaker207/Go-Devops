import { useQuery } from '@tanstack/react-query';
import { bookingApi } from '@/api/booking.api';
import type { AdminOrderListQuery } from '@/types';

export const BOOKING_QUERY_KEY = 'bookings';

export const useAdminOrderList = (query: AdminOrderListQuery) =>
  useQuery({
    queryKey: [BOOKING_QUERY_KEY, query],
    queryFn: () => bookingApi.adminList(query),
    placeholderData: (previous) => previous,
  });

/** Empty id means the drawer is closed; fetch nothing. */
export const useOrderDetail = (id: string | null) =>
  useQuery({
    queryKey: [BOOKING_QUERY_KEY, 'detail', id],
    queryFn: () => bookingApi.detail(id as string),
    enabled: Boolean(id),
  });
