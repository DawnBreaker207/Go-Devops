import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  BasePrice,
  BasePricePayload,
  CreatePricingRulePayload,
  PagedData,
  PricingQuote,
  PricingRule,
  PricingRuleListQuery,
  PublicPriceList,
  UpdatePricingRulePayload,
} from '@/types';

export const pricingApi = {
  publicPrices: () => apiClient.get<ApiResponse<PublicPriceList>>('/pricing').then(unwrap),

  quote: (showtimeId: string, seatType: string) =>
    apiClient
      .get<ApiResponse<PricingQuote>>('/pricing/quote', {
        params: { showtime_id: showtimeId, seat_type: seatType },
      })
      .then(unwrap),

  adminGetBasePrices: () =>
    apiClient.get<ApiResponse<BasePrice[]>>('/admin/pricing/base').then(unwrap),

  adminSetBasePrices: (payload: BasePricePayload) =>
    apiClient.put<ApiResponse<BasePrice[]>>('/admin/pricing/base', payload).then(unwrap),

  adminListRules: (query: PricingRuleListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<PricingRule>>>('/admin/pricing/rules', { params: query })
      .then(unwrap),

  adminGetRule: (id: string) =>
    apiClient.get<ApiResponse<PricingRule>>(`/admin/pricing/rules/${id}`).then(unwrap),

  adminCreateRule: (payload: CreatePricingRulePayload) =>
    apiClient.post<ApiResponse<PricingRule>>('/admin/pricing/rules', payload).then(unwrap),

  adminUpdateRule: (id: string, payload: UpdatePricingRulePayload) =>
    apiClient.patch<ApiResponse<PricingRule>>(`/admin/pricing/rules/${id}`, payload).then(unwrap),

  adminDeleteRule: (id: string) =>
    apiClient.delete<void>(`/admin/pricing/rules/${id}`).then(() => undefined),
};
