import type { Hall, Seat, SeatChange, SeatType } from '@/types';

/** Minimum seat shape shared by admin and customer grid math. */
export interface GridSeat {
  row_label: string;
  col_number: number;
  col_span: number;
  is_gap: boolean;
}

/** Pure grid math, React-free. */
export const SEAT_SIZE = 40;
export const AISLE_WIDTH = 22;

export interface SeatRow<T extends GridSeat = Seat> {
  rowLabel: string;
  seats: T[];
}

// Keep backend row order, never sort: AA comes after Z, not before B.
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

export const parseAisles = (raw: string): number[] =>
  raw
    .split(',')
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((value) => Number.isInteger(value) && value > 0);

// Keep only renderable aisles: 1..seatsPerRow-1, deduped and sorted.
export const normalizeAisles = (aisleAfterCols: number[], seatsPerRow: number): number[] => {
  const seen = new Set<number>();
  aisleAfterCols.forEach((col) => {
    if (Number.isInteger(col) && col >= 1 && col < seatsPerRow) seen.add(col);
  });
  return [...seen].sort((a, b) => a - b);
};

export interface GridLayout {
  templateColumns: string;
  lineOf: (col: number) => number;
  spanOf: (col: number, colSpan: number) => number;
  aisles: number[];
}

// An aisle is its own track, not a margin: margins skew the grid right.
export const buildGridLayout = (seatsPerRow: number, aisleAfterCols: number[]): GridLayout => {
  const aisles = normalizeAisles(aisleAfterCols, seatsPerRow);
  const aisleSet = new Set(aisles);

  const tracks: string[] = [];
  for (let col = 1; col <= seatsPerRow; col += 1) {
    tracks.push(`${SEAT_SIZE}px`);
    if (aisleSet.has(col)) tracks.push(`${AISLE_WIDTH}px`);
  }

  const lineOf = (col: number): number => {
    let aislesBefore = 0;
    aisles.forEach((aisle) => {
      if (aisle < col) aislesBefore += 1;
    });
    return col + aislesBefore;
  };

  const spanOf = (col: number, colSpan: number): number => {
    // A couple swallowing a middle aisle spans 3 tracks, not 2.
    if (colSpan <= 1) return 1;
    return colSpan + (aisleSet.has(col) ? 1 : 0);
  };

  return { templateColumns: tracks.join(' '), lineOf, spanOf, aisles };
};

// Capacity counts like the backend: gaps excluded, a couple is one sellable seat.
export const sellableCapacity = (seats: GridSeat[]): number =>
  seats.reduce((total, seat) => (seat.is_gap ? total : total + 1), 0);

// Backend uppercases but never trims: clean first or a 400 rolls back the whole batch.
export const cleanSeatLabel = (label: string): string => label.trim().toUpperCase();

// Backend cap: 50 changes/call, 500 labels/change.
export const MAX_CHANGES_PER_CALL = 50;
export const MAX_LABELS_PER_CHANGE = 500;

// Chunk to backend caps: one batch is one transaction, oversize fails the whole batch.
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

// Zoom via transform scale, never width/height.
export const ZOOM_MIN = 1;
export const ZOOM_MAX = 1.6;
export const ZOOM_STEP = 0.1;
export const ZOOM_DEFAULT = 1;

export const gridPixelWidth = (templateColumns: string, gapPx: number = 4): number => {
  const tracks = templateColumns.trim().split(/\s+/).filter(Boolean);
  const tracksPx = tracks.reduce((sum, token) => sum + (Number.parseFloat(token) || 0), 0);
  const gaps = Math.max(0, tracks.length - 1) * gapPx;
  return tracksPx + gaps;
};

// Fit the room into view, never upscale past 1.
export const computeFitScale = (
  gridWidthPx: number,
  containerWidthPx: number,
  min: number = ZOOM_MIN,
  max: number = 1
): number => {
  if (gridWidthPx <= 0 || containerWidthPx <= 0) return ZOOM_DEFAULT;
  const raw = containerWidthPx / gridWidthPx;
  return Math.min(max, Math.max(min, raw));
};

export const clampZoom = (value: number, min: number = ZOOM_MIN, max: number = ZOOM_MAX): number =>
  Math.min(max, Math.max(min, value));

// Never trust hall.seats_per_row: no DB constraint, and it has drifted for real.
export const widestColumn = (seats: Pick<GridSeat, 'col_number' | 'col_span'>[]): number =>
  seats.reduce((max, s) => Math.max(max, s.col_number + s.col_span - 1), 0);

// GET /shows/:id/seats omits rows/seats_per_row, so derive from seats.
export const seatsPerRowFromSeats = (seats: Pick<GridSeat, 'col_number' | 'col_span'>[]): number =>
  widestColumn(seats);

export const renderedSeatsPerRow = (hall: Hall, seats: Seat[]): number =>
  Math.max(widestColumn(seats), hall.seats_per_row);

export interface GridSummary {
  gridCells: number;
  seatCount: number;
  sellable: number;
  gaps: number;
  doubleSeats: number;
}

export const summarizeGrid = (hall: Hall, seats: Seat[]): GridSummary => ({
  gridCells: hall.rows * hall.seats_per_row,
  seatCount: seats.length,
  sellable: sellableCapacity(seats),
  gaps: seats.filter((s) => s.is_gap).length,
  doubleSeats: seats.filter((s) => s.col_span === 2).length,
});

export interface GridMismatch {
  actualRows: number;
  actualSeatsPerRow: number;
}

// Mismatch against saved seats, not the draft: a pending row intentionally overshoots rows mid-edit.
export const declaredGridMismatch = (hall: Hall, savedSeats: Seat[]): GridMismatch | null => {
  if (savedSeats.length === 0) return null;
  const actualRows = groupSeatsByRow(savedSeats).length;
  const actualSeatsPerRow = widestColumn(savedSeats);
  if (actualRows === hall.rows && actualSeatsPerRow === hall.seats_per_row) return null;
  return { actualRows, actualSeatsPerRow };
};

// The edit screen is one draft session: every change touches the draft until Save.
export const PENDING_ROW_PREFIX = 'pending-row-';

export const makePendingSeatId = (rowIndex: number, colNumber: number): string =>
  `${PENDING_ROW_PREFIX}${rowIndex}-${colNumber}`;

export const isPendingSeatId = (id: string): boolean => id.startsWith(PENDING_ROW_PREFIX);

export const pendingRowIndexOf = (id: string): number =>
  Number(id.slice(PENDING_ROW_PREFIX.length).split('-')[0]);

export const PENDING_COL_PREFIX = 'pending-col-';

export const makePendingColSeatId = (colIndex: number, rowLabel: string): string =>
  `${PENDING_COL_PREFIX}${colIndex}-${rowLabel}`;

export const isPendingColSeatId = (id: string): boolean => id.startsWith(PENDING_COL_PREFIX);

export const pendingColIndexOf = (id: string): number =>
  Number(id.slice(PENDING_COL_PREFIX.length).split('-')[0]);

// A split's brand-new right seat doesn't exist server-side until Save runs splitSeat.
export const PENDING_SPLIT_PREFIX = 'pending-split-';

export const makeSplitSeatId = (leftSeatId: string): string =>
  `${PENDING_SPLIT_PREFIX}${leftSeatId}`;

export const isPendingSplitSeatId = (id: string): boolean => id.startsWith(PENDING_SPLIT_PREFIX);

export const isUnsavedSeatId = (id: string): boolean =>
  isPendingSeatId(id) || isPendingColSeatId(id) || isPendingSplitSeatId(id);

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

export const buildPendingRow = (
  rowIndex: number,
  seatsPerRow: number,
  rowLabel: string,
  hallId: string,
  totalWidth: number = seatsPerRow
): Seat[] =>
  Array.from({ length: totalWidth }, (_, i) => {
    const col = i + 1;
    return {
      id: makePendingSeatId(rowIndex, col),
      hall_id: hallId,
      label: `${rowLabel}${col}`,
      row_label: rowLabel,
      col_number: col,
      seat_type: 'standard' as SeatType,
      is_gap: col > seatsPerRow,
      col_span: 1 as const,
    };
  });

export const buildPendingColumn = (
  colIndex: number,
  rowLabels: string[],
  colNumber: number,
  hallId: string
): Seat[] =>
  rowLabels.map((rowLabel) => ({
    id: makePendingColSeatId(colIndex, rowLabel),
    hall_id: hallId,
    label: `${rowLabel}${colNumber}`,
    row_label: rowLabel,
    col_number: colNumber,
    seat_type: 'standard' as SeatType,
    is_gap: true,
    col_span: 1 as const,
  }));

export interface SeatPatchGroup {
  labels: string[];
  seat_type?: SeatType;
  is_gap?: boolean;
}

// Diff draft vs saved by id, grouped by type/gap.
export const diffChangedSeats = (saved: Seat[], draft: Seat[]): SeatPatchGroup[] => {
  const savedById = new Map(saved.map((s) => [s.id, s]));
  const groups = new Map<string, SeatPatchGroup>();
  draft.forEach((seat) => {
    if (isUnsavedSeatId(seat.id)) return;
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

export const countSeatsByType = (seats: Seat[]): Record<SeatType, number> => {
  const counts: Record<SeatType, number> = { standard: 0, vip: 0, couple: 0, recliner: 0 };
  seats.forEach((seat) => {
    if (seat.is_gap) return;
    counts[seat.seat_type] += 1;
  });
  return counts;
};

// Mergeable pair: same row, adjacent, single, non-gap (mirrors BE spanSet).
export const areAdjacentSeats = (a: Seat, b: Seat): boolean =>
  a.row_label === b.row_label &&
  a.col_span === 1 &&
  b.col_span === 1 &&
  !a.is_gap &&
  !b.is_gap &&
  Math.abs(a.col_number - b.col_number) === 1;

export interface MergeOp {
  leftLabel: string;
  rightLabel: string;
}

export interface SplitOp {
  label: string;
}

/** Merge/split trong phien draft, doi chieu voi server theo label (khong theo id vi merge xoa seat phai, split de id moi). Baseline gom ca row/col vua tao trong cung lan Save. */
export const diffMergeSplitOps = (
  baseline: Pick<Seat, 'label' | 'row_label' | 'col_number' | 'col_span'>[],
  draft: Pick<Seat, 'label' | 'row_label' | 'col_number' | 'col_span'>[]
): { merges: MergeOp[]; splits: SplitOp[] } => {
  const baselineByLabel = new Map(baseline.map((s) => [s.label, s]));
  const draftByLabel = new Map(draft.map((s) => [s.label, s]));

  const merges: MergeOp[] = draft
    .filter((seat) => seat.col_span === 2 && baselineByLabel.get(seat.label)?.col_span !== 2)
    .map((seat) => ({
      leftLabel: seat.label,
      rightLabel: `${seat.row_label}${seat.col_number + 1}`,
    }));

  const splits: SplitOp[] = baseline
    .filter((seat) => {
      if (seat.col_span !== 2) return false;
      const after = draftByLabel.get(seat.label);
      return after !== undefined && after.col_span !== 2;
    })
    .map((seat) => ({ label: seat.label }));

  return { merges, splits };
};
