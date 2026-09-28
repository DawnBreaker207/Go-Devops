import type { PageQuery } from './api';

// Mirrors Go DTO internal/dto/discount.go.

export type DiscountKind = 'percent' | 'amount';

export const DISCOUNT_KINDS: readonly DiscountKind[] = ['percent', 'amount'] as const;

export interface DiscountApplied {
  booking_id: string;
  code?: string;
  subtotal: number;
  discount: number;
  payable: number;
}

export interface DiscountCode {
  id: string;
  code: string;
  description?: string;
  kind: DiscountKind;
  value: number;
  max_discount?: number;
  min_order: number;
  starts_at?: string;
  ends_at?: string;
  max_uses?: number;
  used_count: number;
  active: boolean;
  created_at: string;
}

export interface DiscountListQuery extends PageQuery {
  active?: boolean;
}

export interface CreateDiscountPayload {
  code: string;
  description?: string;
  kind: DiscountKind;
  value: number;
  max_discount?: number;
  min_order?: number;
  starts_at?: string;
  ends_at?: string;
  max_uses?: number;
  active?: boolean;
}

export interface UpdateDiscountPayload {
  description?: string;
  value?: number;
  max_discount?: number;
  min_order?: number;
  starts_at?: string;
  ends_at?: string;
  max_uses?: number;
  active?: boolean;
}
