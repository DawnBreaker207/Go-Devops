import { useMutation, useQueryClient } from '@tanstack/react-query';
import { discountApi } from '@/api/discount.api';
import { ORDER_QUERY_KEY } from './useOrders';

/** Applying or removing a code changes what the order owes, so EVERY order query
 *  must be refetched: `payable_amount` on the order is the authoritative number
 *  the sidebar and the pay button read, not anything held in component state.
 *  Invalidating the whole key covers detail, status and the list at once. */
const useInvalidateOrder = () => {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: [ORDER_QUERY_KEY] });
};

export const useApplyDiscount = (bookingId: string) => {
  const invalidate = useInvalidateOrder();
  return useMutation({
    mutationFn: (code: string) => discountApi.apply(bookingId, code),
    onSuccess: () => void invalidate(),
  });
};

export const useRemoveDiscount = (bookingId: string) => {
  const invalidate = useInvalidateOrder();
  return useMutation({
    mutationFn: () => discountApi.remove(bookingId),
    onSuccess: () => void invalidate(),
  });
};
