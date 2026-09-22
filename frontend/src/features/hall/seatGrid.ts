import type { Hall, Seat, SeatChange, SeatType } from '@/types';

/** Minimum seat shape for grid math - `Seat` (admin) and `SeatMapSeat` (customer) both satisfy it, so all math is shared. */
export interface GridSeat {
  row_label: string;
  col_number: number;
  col_span: number;
  is_gap: boolean;
}

/** Seat-grid math, React-free (easiest to get wrong, hence tested). Couple seats swallow a column (never render by index); `aisle_after_cols` is length-checked only, so filter here. */

/** One seat cell, in px. */
export const SEAT_SIZE = 40;
/** Aisle gap between two columns. */
export const AISLE_WIDTH = 22;

export interface SeatRow<T extends GridSeat = Seat> {
  rowLabel: string;
  seats: T[];
}

/** Keeps backend row order - never sort by row_label (AA precedes B alphabetically but is row 27). */
export const groupSeatsByRow = <T extends GridSeat>(seats: T[]): SeatRow<T>[] => {
  const rows: SeatRow<T>[] = [];
  const index = new Map<string, SeatRow<T>>();
  seats.forEach((seat) => {
    let row = index.get(seat.row_label);
    if (!row) {
      row = { rowLabel: seat.row_label, seats: [] };
      index.set(seat.row_label, row);
      rows.push(row);
    }
    row.seats.push(seat);
  });
  return rows;
};

/** Keep only renderable aisles: 1..seatsPerRow-1, deduped (past the last column renders nothing). */
export const normalizeAisles = (aisleAfterCols: number[], seatsPerRow: number): number[] => {
  const seen = new Set<number>();
  aisleAfterCols.forEach((col) => {
    if (Number.isInteger(col) && col >= 1 && col < seatsPerRow) seen.add(col);
  });
  return [...seen].sort((a, b) => a - b);
};

export interface GridLayout {
  /** Value for CSS `grid-template-columns`. */
  templateColumns: string;
  /** Grid line a column starts at (CSS grids count from 1). */
  lineOf: (col: number) => number;
  /** TRACKS one seat spans - not col_span when an aisle track sits between. */
  spanOf: (col: number, colSpan: number) => number;
  aisles: number[];
}

/** Aisles are their OWN TRACK, not seat margin (margin squeezes cells, skewing the grid right). */
export const buildGridLayout = (seatsPerRow: number, aisleAfterCols: number[]): GridLayout => {
  const aisles = normalizeAisles(aisleAfterCols, seatsPerRow);
  const aisleSet = new Set(aisles);

  const tracks: string[] = [];
  for (let col = 1; col <= seatsPerRow; col += 1) {
    tracks.push(`${SEAT_SIZE}px`);
    if (aisleSet.has(col)) tracks.push(`${AISLE_WIDTH}px`);
  }

  const lineOf = (col: number): number => {
    // Line = 1 + tracks before it = 1 + (col-1) cells + inserted aisles.
    let aislesBefore = 0;
    aisles.forEach((aisle) => {
      if (aisle < col) aislesBefore += 1;
    });
    return col + aislesBefore;
  };

  const spanOf = (col: number, colSpan: number): number => {
    // Couple covers col and col+1. An aisle between stretches it over the aisle track too: 3 tracks, not 2.
    if (colSpan <= 1) return 1;
    return colSpan + (aisleSet.has(col) ? 1 : 0);
  };

  return { templateColumns: tracks.join(' '), lineOf, spanOf, aisles };
};

/** Sellable capacity, counted like the backend: `rows * seats_per_row` counts cells (couples swallow a column, gaps are unsellable). */
export const sellableCapacity = (seats: GridSeat[]): number =>
  seats.reduce((total, seat) => (seat.is_gap ? total : total + 1), 0);

/** Backend uppercases labels but does NOT trim (400 ROLLS BACK the whole batch) - clean here first. */
export const cleanSeatLabel = (label: string): string => label.trim().toUpperCase();

/** Backend cap: 50 changes/call, 500 labels/change. */
export const MAX_CHANGES_PER_CALL = 50;
export const MAX_LABELS_PER_CHANGE = 500;

/** Chunk labels within backend caps (one batch = ONE transaction, oversized batches fail whole). */
export const chunkLabels = (labels: string[]): string[][][] => {
  const changes: string[][] = [];
  for (let i = 0; i < labels.length; i += MAX_LABELS_PER_CHANGE) {
    changes.push(labels.slice(i, i + MAX_LABELS_PER_CHANGE));
  }
  const batches: string[][][] = [];
  for (let i = 0; i < changes.length; i += MAX_CHANGES_PER_CALL) {
    batches.push(changes.slice(i, i + MAX_CHANGES_PER_CALL));
  }
  return batches;
};

/** Real column count from seat data (never trust `hall.seats_per_row`: no DB constraint ties it to `seats`, drifted for real once). */
export const widestColumn = (seats: Pick<GridSeat, 'col_number' | 'col_span'>[]): number =>
  seats.reduce((max, s) => Math.max(max, s.col_number + s.col_span - 1), 0);

/** Customer variant: `GET /shows/:id/seats` omits rows/seats_per_row - derive from seats. */
export const seatsPerRowFromSeats = (seats: Pick<GridSeat, 'col_number' | 'col_span'>[]): number =>
  widestColumn(seats);

export const renderedSeatsPerRow = (hall: Hall, seats: Seat[]): number =>
  Math.max(widestColumn(seats), hall.seats_per_row);

/** Grid summary for the screen's recap panel. */
export interface GridSummary {
  gridCells: number;
  seatCount: number;
  sellable: number;
  gaps: number;
  doubleSeats: number;
  /** true when the hall's declared rows/seats_per_row mismatch real seats. */
  declaredMismatch: boolean;
  actualRows: number;
  actualSeatsPerRow: number;
}

export const summarizeGrid = (hall: Hall, seats: Seat[]): GridSummary => {
  const actualRows = groupSeatsByRow(seats).length;
  const actualSeatsPerRow = widestColumn(seats);
  return {
    gridCells: hall.rows * hall.seats_per_row,
    seatCount: seats.length,
    sellable: sellableCapacity(seats),
    gaps: seats.filter((s) => s.is_gap).length,
    doubleSeats: seats.filter((s) => s.col_span === 2).length,
    // A seatless hall is not "mismatched" - different state.
    declaredMismatch:
      seats.length > 0 && (actualRows !== hall.rows || actualSeatsPerRow !== hall.seats_per_row),
    actualRows,
    actualSeatsPerRow,
  };
};

/** Edit screen is ONE DRAFT SESSION: changes touch local draft only; one Save folds everything (new rows get fake `pending-row-N-col` ids). */
export const PENDING_ROW_PREFIX = 'pending-row-';

export const makePendingSeatId = (rowIndex: number, colNumber: number): string =>
  `${PENDING_ROW_PREFIX}${rowIndex}-${colNumber}`;

export const isPendingSeatId = (id: string): boolean => id.startsWith(PENDING_ROW_PREFIX);

export const pendingRowIndexOf = (id: string): number =>
  Number(id.slice(PENDING_ROW_PREFIX.length).split('-')[0]);

/** Go dto.RowLabel port: base-26 Excel-style columns (A..Z, AA, AB...). */
export const rowLabelFromIndex = (n: number): string => {
  let label = '';
  let value = n;
  while (value > 0) {
    value -= 1;
    label = String.fromCharCode(65 + (value % 26)) + label;
    value = Math.floor(value / 26);
  }
  return label;
};

/** Standard-seat placeholder for a pending row. The label is PREVIEW only (server may assign another); Save matches by column order. */
export const buildPendingRow = (
  rowIndex: number,
  seatsPerRow: number,
  rowLabel: string,
  hallId: string
): Seat[] =>
  Array.from({ length: seatsPerRow }, (_, i) => {
    const col = i + 1;
    return {
      id: makePendingSeatId(rowIndex, col),
      hall_id: hallId,
      label: `${rowLabel}${col}`,
      row_label: rowLabel,
      col_number: col,
      seat_type: 'standard' as SeatType,
      is_gap: false,
      col_span: 1 as const,
    };
  });

export interface SeatPatchGroup {
  labels: string[];
  seat_type?: SeatType;
  is_gap?: boolean;
}

/** Diff real (non-pending) seats draft-vs-saved, grouped by (seat_type, is_gap) - id-matched even right after a pending save. */
export const diffChangedSeats = (saved: Seat[], draft: Seat[]): SeatPatchGroup[] => {
  const savedById = new Map(saved.map((s) => [s.id, s]));
  const groups = new Map<string, SeatPatchGroup>();
  draft.forEach((seat) => {
    if (isPendingSeatId(seat.id)) return;
    const before = savedById.get(seat.id);
    if (!before || (before.seat_type === seat.seat_type && before.is_gap === seat.is_gap)) return;
    const key = `${seat.seat_type}|${seat.is_gap}`;
    let group = groups.get(key);
    if (!group) {
      group = { labels: [], seat_type: seat.seat_type, is_gap: seat.is_gap };
      groups.set(key, group);
    }
    group.labels.push(cleanSeatLabel(seat.label));
  });
  return [...groups.values()];
};

/** Like chunkLabels for multi-patch Saves, same two backend caps. */
export const buildSeatChangeBatches = (groups: SeatPatchGroup[]): SeatChange[][] => {
  const changes: SeatChange[] = [];
  groups.forEach((group) => {
    for (let i = 0; i < group.labels.length; i += MAX_LABELS_PER_CHANGE) {
      changes.push({
        selector: { labels: group.labels.slice(i, i + MAX_LABELS_PER_CHANGE) },
        seat_type: group.seat_type,
        is_gap: group.is_gap,
      });
    }
  });
  const batches: SeatChange[][] = [];
  for (let i = 0; i < changes.length; i += MAX_CHANGES_PER_CALL) {
    batches.push(changes.slice(i, i + MAX_CHANGES_PER_CALL));
  }
  return batches;
};

/** Mergeable pair: same row, adjacent, single, non-gap (mirrors BE `spanSet`). */
export const areAdjacentSeats = (a: Seat, b: Seat): boolean =>
  a.row_label === b.row_label &&
  a.col_span === 1 &&
  b.col_span === 1 &&
  !a.is_gap &&
  !b.is_gap &&
  Math.abs(a.col_number - b.col_number) === 1;
