import { useMutation, useQuery } from '@tanstack/react-query';
import { comboApi } from '@/api/combo.api';
import type { CreateComboOrderPayload } from '@/types';

export const COMBO_QUERY_KEY = 'combos';

/** GET /combos - public. Empty list or error never blocks the ticket flow, so no aggressive retry here. */
export const useCombos = () =>
  useQuery({
    queryKey: [COMBO_QUERY_KEY],
    queryFn: () => comboApi.list(),
    staleTime: 60_000,
  });

/** POST /combo-orders - independent of /orders/hold, invalidates nothing of the ticket flow. Its errors must never block checkout. */
export const useCreateComboOrder = () =>
  useMutation({
    mutationFn: (payload: CreateComboOrderPayload) => comboApi.createOrder(payload),
  });
