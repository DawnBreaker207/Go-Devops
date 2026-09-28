import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { comboApi } from '@/api/combo.api';
import type { ComboOrder, CreateComboOrderPayload } from '@/types';

export const COMBO_QUERY_KEY = 'combos';

/** GET /combos - public. Empty list or error never blocks the ticket flow, so no aggressive retry here. */
export const useCombos = () =>
  useQuery({
    queryKey: [COMBO_QUERY_KEY],
    queryFn: () => comboApi.list(),
    staleTime: 60_000,
  });

/** POST /combo-orders - independent of /orders/hold; errors must never block checkout. Invalidates the combo-order list so the checkout summary picks it up. */
export const useCreateComboOrder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateComboOrderPayload) => comboApi.createOrder(payload),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: [COMBO_ORDER_QUERY_KEY] }),
  });
};

export const COMBO_ORDER_QUERY_KEY = 'combo-orders';

/** The combo order attached to THIS booking, read from the server (survives reload) and matched by `booking_id` since /combo-orders/me is the full history; one page suffices since it was created seconds ago. */
export const useComboOrderForBooking = (bookingId: string | undefined) => {
  const query = useQuery({
    queryKey: [COMBO_ORDER_QUERY_KEY, { page: 1, page_size: 10 }],
    queryFn: () => comboApi.myOrders({ page: 1, page_size: 10 }),
    enabled: Boolean(bookingId),
  });
  const found: ComboOrder | null =
    query.data?.items.find((order) => order.booking_id === bookingId) ?? null;
  return { ...query, comboOrder: found };
};
