import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { orderApi } from '@/api/order.api';
import type { HoldPayload, PageQuery, PayPayload } from '@/types';

export const ORDER_QUERY_KEY = 'orders';

export const useMyOrders = (query: PageQuery) =>
  useQuery({
    queryKey: [ORDER_QUERY_KEY, query],
    queryFn: () => orderApi.list(query),
    placeholderData: (previous) => previous,
  });

export const useOrderDetail = (bookingId: string | undefined) =>
  useQuery({
    queryKey: [ORDER_QUERY_KEY, 'detail', bookingId],
    queryFn: () => orderApi.detail(bookingId as string),
    enabled: Boolean(bookingId),
  });

/**
 * Do trang thai don trong luc cho cong thanh toan bao ve.
 *
 * Backend khong day gi ve phia client cho viec nay (SSE chi phat trang thai
 * GHE cua mot suat chieu, khong phat trang thai DON), nen phai hoi lai. Chi
 * bat khi don con `pending`: don da chot hoac da het han thi khong doi nua.
 */
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
