import type { PageQuery } from './api';
import type { SeatType } from './hall';

// Mirrors Go DTO internal/dto/pricing.go.
export interface BasePrice {
  seat_type: SeatType;
  price: number;
  updated_at: string;
}

export interface BasePricePayload {
  prices: Partial<Record<SeatType, number>>;
}

export type PricingAdjustKind = 'percent' | 'fixed';

export const PRICING_ADJUST_KINDS: readonly PricingAdjustKind[] = ['percent', 'fixed'] as const;

export interface PricingRule {
  id: string;
  name: string;
  day_of_week?: number;
  start_time?: string;
  end_time?: string;
  specific_date?: string;
  adjust_kind: PricingAdjustKind;
  adjust_value: number;
  priority: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface PricingRuleListQuery extends PageQuery {
  active?: boolean;
}

export interface CreatePricingRulePayload {
  name: string;
  day_of_week?: number;
  start_time?: string;
  end_time?: string;
  specific_date?: string;
  adjust_kind: PricingAdjustKind;
  adjust_value: number;
  priority?: number;
  active?: boolean;
}

export interface UpdatePricingRulePayload {
  name?: string;
  day_of_week?: number;
  start_time?: string;
  end_time?: string;
  specific_date?: string;
  adjust_kind?: PricingAdjustKind;
  adjust_value?: number;
  priority?: number;
  active?: boolean;
}

export interface PublicPriceList {
  from_price: number;
  prices: Record<SeatType, number>;
}

export interface AppliedPricingRule {
  rule_id: string;
  name: string;
  adjust_kind: PricingAdjustKind;
  adjust_value: number;
}

export interface PricingQuote {
  base: number;
  applied: AppliedPricingRule[];
  final: number;
}
