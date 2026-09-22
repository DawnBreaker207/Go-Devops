import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { discountApi } from '@/api/discount.api';
import type { CreateDiscountPayload, DiscountListQuery, UpdateDiscountPayload } from '@/types';

export const DISCOUNT_QUERY_KEY = 'discounts';

export const useDiscountList = (query: DiscountListQuery) =>
  useQuery({
    queryKey: [DISCOUNT_QUERY_KEY, query],
    queryFn: () => discountApi.adminList(query),
    placeholderData: (previous) => previous,
  });

const useInvalidate = () => {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: [DISCOUNT_QUERY_KEY] });
};

export const useCreateDiscount = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (payload: CreateDiscountPayload) => discountApi.adminCreate(payload),
    onSuccess: () => void invalidate(),
  });
};

export const useUpdateDiscount = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateDiscountPayload }) =>
      discountApi.adminUpdate(id, payload),
    onSuccess: () => void invalidate(),
  });
};

export const useDeleteDiscount = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (id: string) => discountApi.adminDelete(id),
    onSuccess: () => void invalidate(),
  });
};
