/** Mirror of internal/dto/hall.go. Optionality rule: Go value fields without `omitempty` are required; with `omitempty` or pointers are optional. Top traps: prices read as ARRAY but write as MAP (different orders); seats are a bare unpaginated array; grid cells != rows * seats_per_row with col_span=2. */

/** No endpoint returns this list; FE owns it. */
export type SeatType = 'standard' | 'vip' | 'couple' | 'recliner';

/** models.AllSeatTypes order; also the PUT /prices return order. GET /prices sorts alphabetically instead, so every display re-sorts by this constant. */
export const SEAT_TYPES: readonly SeatType[] = ['standard', 'vip', 'couple', 'recliner'] as const;

export type ScreenPosition = 'front' | 'back';
export const SCREEN_POSITIONS: readonly ScreenPosition[] = ['front', 'back'] as const;

export type HallTemplateName = 'small' | 'medium' | 'large';

/** SQL: col_span SMALLINT CHECK (col_span IN (1,2)). */
export type ColSpan = 1 | 2;

/** No field omitempty. */
export interface Hall {
  id: string;
  name: string;
  rows: number;
  seats_per_row: number;
  screen_position: ScreenPosition;
  /** JSONB NOT NULL DEFAULT '[]'; backend only checks length <= 49, never values, so 0/negative/over-column entries are possible. Hint only; backend computes nothing from it. */
  aisle_after_cols: number[];
  /** active=false halls are rejected for new showtimes (409). */
  active: boolean;
  created_at: string;
  updated_at: string;
}

/** dto.SeatResponse. `label` is composed by the backend (row_label + col_number), not stored, not writable. The real grid key is (row_label, col_number). */
export interface Seat {
  id: string;
  hall_id: string;
  label: string;
  row_label: string;
  col_number: number;
  seat_type: SeatType;
  /** Grid cell exists but isn't sellable (aisle, wheelchair, pillar). Still priced with a showtime_seats row; holding it fails ErrSeatNotSellable. */
  is_gap: boolean;
  /** col_span=2 occupies N and N+1 with NO seat record at N+1, so column numbers have holes; never render by array index. */
  col_span: ColSpan;
}

/** Read side (array) and PUT echo (same array). */
export interface HallPrice {
  seat_type: SeatType;
  /** int64 whole VND. 0 = NOT CONFIGURED, not free. */
  price: number;
}

/** Bare array of 3. */
export interface HallTemplate {
  name: HallTemplateName;
  rows: number;
  seats_per_row: number;
  /** Sellable count (excludes gaps, counts col_span=2 once); below rows * seats_per_row. */
  seat_count: number;
  /** SPARSE: seat types with 0 seats are missing from the map. */
  seat_count_by_type: Partial<Record<SeatType, number>>;
}

/** Shared by POST /admin/halls AND PUT .../layout. On /layout, `name` and `prices` are still binding-required but ignored by the service. */
export interface HallPayload {
  name: string;
  template?: HallTemplateName;
  rows?: number;
  seats_per_row?: number;
  /** {seat type: 1-based ROW NUMBERS as strings}, e.g. {"vip": ["6","7"]} = rows F,G. Row numbers, not labels. Unlisted rows are standard. */
  seat_types?: Partial<Record<SeatType, string[]>>;
  /** Seat labels turned into gaps, e.g. ["D5","D6"]. Max 200. */
  gaps?: string[];
  /** Anchor labels for 2-col couple seats, e.g. ["D3"] = D3-D4. Max 100. */
  spans?: string[];
  screen_position?: ScreenPosition;
  aisle_after_cols?: number[];
  /** REQUIRED. POST wants all 4 types with price > 0; PUT /layout ignores it entirely. */
  prices: Record<SeatType, number>;
}

/** Meta-only update; empty keeps current. Never touches seats. Omit to keep; send [] to CLEAR all aisles (not a Go pointer). */
export interface UpdateHallPayload {
  name?: string;
  screen_position?: ScreenPosition;
  /** Not a Go pointer: omit to keep, send [] to CLEAR all aisles. */
  aisle_after_cols?: number[];
  active?: boolean;
}

/** dto.CloneHallRequest - POST /admin/halls/:id/clone. */
export interface CloneHallPayload {
  name: string;
  /** No binding tag, defaults false. false = new hall gets NO prices. */
  copy_prices?: boolean;
}

/** Write side is a MAP with all 4 types. */
export interface PricePayload {
  prices: Record<SeatType, number>;
}

/** Exactly one selector field must be non-empty. Go binding can't catch it (value struct); the service returns 400/40001 details {selector: "exactly one of ..."}. */
export interface SeatSelector {
  /** Max 500. Uppercased for matching but NOT trimmed: a stray space matches nothing and breaks the batch. Trim in FE. */
  labels?: string[];
  /** Max 50; trimmed AND uppercased. */
  rows?: string[];
  /** Max 50 exact column numbers. */
  cols?: number[];
  /** Inclusive rectangle, e.g. "A1:C4". */
  range?: string;
}

/** Missing both seat_type and is_gap is still valid: an empty change. */
export interface SeatChange {
  selector: SeatSelector;
  seat_type?: SeatType;
  is_gap?: boolean;
}

/** 1..50 changes. */
export interface BulkSeatUpdatePayload {
  changes: SeatChange[];
}

/** dto.SeatUpdateRequest - PUT /admin/halls/:id/seats/:seatId. */
export interface SeatUpdatePayload {
  seat_type?: SeatType;
  is_gap?: boolean;
}

/** dto.MergeSeatsRequest - POST /admin/halls/:id/seats/merge. Right seat is
 *  deleted entirely; left seat becomes col_span=2, seat_type=couple. */
export interface MergeSeatsPayload {
  left_label: string;
  right_label: string;
}

/** dto.SplitSeatRequest - POST /admin/halls/:id/seats/split. */
export interface SplitSeatPayload {
  label: string;
}
