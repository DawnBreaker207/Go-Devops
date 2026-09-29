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

export type ComboPayMethod = 'cash' | 'pos';

export interface CounterComboOrderPayload {
  items: ComboOrderItemPayload[];
  pay_method: ComboPayMethod;
  customer_name?: string;
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
  sold_channel: string;
  pay_method?: string;
  customer_name?: string;
  items: ComboOrderItem[];
  created_at: string;
}

export interface ComboPickupItem {
  combo_id: string;
  combo_name: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
}

export interface ComboPickup {
  order_id: string;
  customer_name: string;
  customer_email: string;
  booking_id?: string;
  movie_title?: string;
  showtime_at?: string;
  total: number;
  items: ComboPickupItem[];
  created_at: string;
}

export interface PickupQuery {
  date?: string;
  search?: string;
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
