// Mirrors Go DTO internal/dto/hall.go.

export type SeatType = 'standard' | 'vip' | 'couple' | 'recliner';

export const SEAT_TYPES: readonly SeatType[] = ['standard', 'vip', 'couple', 'recliner'] as const;

export type ScreenPosition = 'front' | 'back';
export const SCREEN_POSITIONS: readonly ScreenPosition[] = ['front', 'back'] as const;

export type HallTemplateName = 'small' | 'medium' | 'large';

export type ColSpan = 1 | 2;

export interface Hall {
  id: string;
  name: string;
  rows: number;
  seats_per_row: number;
  screen_position: ScreenPosition;
  aisle_after_cols: number[];
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Seat {
  id: string;
  hall_id: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  is_gap: boolean;
  col_span: ColSpan;
}

export interface HallTemplate {
  name: HallTemplateName;
  rows: number;
  seats_per_row: number;
  seat_count: number;
  seat_count_by_type: Partial<Record<SeatType, number>>;
}

export interface HallPayload {
  name: string;
  template?: HallTemplateName;
  rows?: number;
  seats_per_row?: number;
  seat_types?: Partial<Record<SeatType, string[]>>;
  gaps?: string[];
  spans?: string[];
  screen_position?: ScreenPosition;
  aisle_after_cols?: number[];
  active?: boolean;
}

export interface UpdateHallPayload {
  name?: string;
  screen_position?: ScreenPosition;
  aisle_after_cols?: number[];
  active?: boolean;
}

export interface CloneHallPayload {
  name: string;
}

export interface SeatSelector {
  labels?: string[];
  rows?: string[];
  cols?: number[];
  range?: string;
}

export interface SeatChange {
  selector: SeatSelector;
  seat_type?: SeatType;
  is_gap?: boolean;
}

export interface BulkSeatUpdatePayload {
  changes: SeatChange[];
}

export interface SeatUpdatePayload {
  seat_type?: SeatType;
  is_gap?: boolean;
}

export interface MergeSeatsPayload {
  left_label: string;
  right_label: string;
}

export interface SplitSeatPayload {
  label: string;
}
