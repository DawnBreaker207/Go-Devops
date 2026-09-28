import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { pricingApi } from '@/api/pricing.api';
import type {
  BasePricePayload,
  CreatePricingRulePayload,
  PricingRuleListQuery,
  UpdatePricingRulePayload,
} from '@/types';

export const PRICING_QUERY_KEY = 'pricing';

const useInvalidate = () => {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: [PRICING_QUERY_KEY] });
};

export const useBasePrices = () =>
  useQuery({
    queryKey: [PRICING_QUERY_KEY, 'base'],
    queryFn: () => pricingApi.adminGetBasePrices(),
  });

export const useSetBasePrices = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (payload: BasePricePayload) => pricingApi.adminSetBasePrices(payload),
    onSuccess: () => void invalidate(),
  });
};

export const usePricingRules = (query: PricingRuleListQuery) =>
  useQuery({
    queryKey: [PRICING_QUERY_KEY, 'rules', query],
    queryFn: () => pricingApi.adminListRules(query),
    placeholderData: (previous) => previous,
  });

export const useCreatePricingRule = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (payload: CreatePricingRulePayload) => pricingApi.adminCreateRule(payload),
    onSuccess: () => void invalidate(),
  });
};

export const useUpdatePricingRule = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdatePricingRulePayload }) =>
      pricingApi.adminUpdateRule(id, payload),
    onSuccess: () => void invalidate(),
  });
};

export const useDeletePricingRule = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (id: string) => pricingApi.adminDeleteRule(id),
    onSuccess: () => void invalidate(),
  });
};
