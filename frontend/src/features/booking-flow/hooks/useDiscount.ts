import { useMutation, useQueryClient } from '@tanstack/react-query';
import { discountApi } from '@/api/discount.api';
import { ORDER_QUERY_KEY } from './useOrders';

/** Applying/removing a code changes `payable_amount`, so the whole order query key must be refetched. */
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
