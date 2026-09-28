import type { PageQuery } from './api';

// Mirrors Go DTO internal/dto/combo.go.

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
  quantity: number;
}

export interface CreateComboOrderPayload {
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

export interface AdminComboListQuery extends PageQuery {
  active?: boolean;
}

export interface CreateComboPayload {
  name: string;
  description?: string;
  price: number;
  image_url?: string;
  active?: boolean;
}

export interface UpdateComboPayload {
  name?: string;
  description?: string;
  price?: number;
  image_url?: string;
  active?: boolean;
}
