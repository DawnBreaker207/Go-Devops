import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { orderApi } from '@/api/order.api';
import type { HoldPayload, PageQuery, PayPayload } from '@/types';

export const ORDER_QUERY_KEY = 'orders';

export const useMyOrders = (query: PageQuery, enabled = true) =>
  useQuery({
    queryKey: [ORDER_QUERY_KEY, query],
    queryFn: () => orderApi.list(query),
    placeholderData: (previous) => previous,
    enabled,
  });

export const useOrderDetail = (bookingId: string | undefined) =>
  useQuery({
    queryKey: [ORDER_QUERY_KEY, 'detail', bookingId],
    queryFn: () => orderApi.detail(bookingId as string),
    enabled: Boolean(bookingId),
  });

/** Order status while awaiting the gateway. No server push for orders (SSE covers seats only), so poll - pending only. */
export const useOrderStatus = (bookingId: string | undefined, polling: boolean) =>
  useQuery({
    queryKey: [ORDER_QUERY_KEY, 'status', bookingId],
    queryFn: () => orderApi.status(bookingId as string),
    enabled: Boolean(bookingId),
    refetchInterval: polling ? 4000 : false,
  });

export const usePaymentProviders = () =>
  useQuery({
    queryKey: ['payment-providers'],
    queryFn: () => orderApi.providers(),
    staleTime: Infinity,
  });

export const useHoldSeats = () =>
  useMutation({ mutationFn: (payload: HoldPayload) => orderApi.hold(payload) });

/** Open an empty order at entry + silent refresh heartbeat (auto-refetches the extended order). */
export const useInitOrder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (showId: string) => orderApi.init(showId),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: [ORDER_QUERY_KEY, 'detail', res.booking_id] });
    },
  });
};

export const useRefreshOrder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (bookingId: string) => orderApi.refresh(bookingId),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: [ORDER_QUERY_KEY, 'detail', res.booking_id] });
    },
  });
};

export const usePayOrder = () =>
  useMutation({
    mutationFn: ({ bookingId, payload }: { bookingId: string; payload: PayPayload }) =>
      orderApi.pay(bookingId, payload),
  });

export const useConfirmOrder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (bookingId: string) => orderApi.confirm(bookingId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [ORDER_QUERY_KEY] }),
  });
};

export const useCancelOrder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (bookingId: string) => orderApi.cancel(bookingId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [ORDER_QUERY_KEY] }),
  });
};
