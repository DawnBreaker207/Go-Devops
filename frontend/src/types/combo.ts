import type { PageQuery } from './api';

/** Concessions, mirroring internal/dto/combo.go. Fully independent from booking: `booking_id` is an optional "pick up together" link; failures on either side must not affect the other. */

/** Public bare array, unpaginated. */
export interface Combo {
  id: string;
  name: string;
  description?: string;
  /** int64 whole VND. */
  price: number;
  image_url?: string;
  active: boolean;
}

export interface ComboOrderItemPayload {
  combo_id: string;
  /** 1-20, enforced in CreateComboOrderRequest. */
  quantity: number;
}

/** Requires customer role. */
export interface CreateComboOrderPayload {
  /** Optional: link to an in-progress booking for joint pickup. */
  booking_id?: string;
  items: ComboOrderItemPayload[];
}

export interface ComboOrderItem {
  combo_id: string;
  combo_name: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
}

export interface ComboOrder {
  id: string;
  booking_id?: string;
  status: string;
  total: number;
  items: ComboOrderItem[];
  created_at: string;
}

/* --- Operator catalogue: /admin/concessions (admin AND staff, like halls) --- */

/** Paged, and unlike the public list it INCLUDES inactive products. */
export interface AdminComboListQuery extends PageQuery {
  /** Go pointer on the backend: absent means no filter, distinct from false. */
  active?: boolean;
}

/** POST /admin/concessions. `active` omitted defaults to TRUE on the backend. */
export interface CreateComboPayload {
  name: string;
  description?: string;
  /** int64 whole VND; 0 is allowed and means a giveaway, not "unset". */
  price: number;
  image_url?: string;
  active?: boolean;
}

/** PATCH /admin/concessions/:id - PARTIAL. An omitted field is left alone; a body
 *  with no field at all is 400/40001 "nothing to update". */
export interface UpdateComboPayload {
  name?: string;
  description?: string;
  price?: number;
  image_url?: string;
  active?: boolean;
}
