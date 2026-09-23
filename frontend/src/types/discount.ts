import type { PageQuery } from './api';

/** Discount codes, mirroring internal/dto/discount.go.
 *
 *  The key fact for any screen using these: `subtotal` is the seat total and
 *  never changes when a code is applied; `payable` is what the gateway charges.
 *  The backend keeps them apart on purpose — a discounted `total_amount` would
 *  break the seat-price assertion that runs when an order is confirmed. */

/** No endpoint returns this list; the frontend owns it, like SeatType. */
export type DiscountKind = 'percent' | 'amount';

export const DISCOUNT_KINDS: readonly DiscountKind[] = ['percent', 'amount'] as const;

/** POST/DELETE /orders/:id/discount both answer this shape. */
export interface DiscountApplied {
  booking_id: string;
  /** Absent after a removal. */
  code?: string;
  /** Seat total, unchanged by the discount. */
  subtotal: number;
  discount: number;
  /** subtotal - discount. */
  payable: number;
}

/* --- Operator catalogue: /admin/discounts, ADMIN-ONLY (unlike concessions) --- */

export interface DiscountCode {
  id: string;
  code: string;
  description?: string;
  kind: DiscountKind;
  /** Percent (1..100) for `percent`, whole VND for `amount`. */
  value: number;
  /** Caps a percentage code; absent for `amount` (the backend rejects it there). */
  max_discount?: number;
  /** Subtotal required before the code applies; 0 = no minimum. */
  min_order: number;
  starts_at?: string;
  ends_at?: string;
  /** Absent = unlimited. */
  max_uses?: number;
  used_count: number;
  active: boolean;
  created_at: string;
}

export interface DiscountListQuery extends PageQuery {
  /** Go pointer: absent means no filter, distinct from false. */
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
  /** Omitted defaults to TRUE. */
  active?: boolean;
}

/** PARTIAL. `code` and `kind` are absent on purpose: the backend refuses to
 *  change either, because it would rewrite what the code meant for orders that
 *  already used it. */
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
