import { describe, expect, it } from 'vitest';
import {
  AISLE_WIDTH,
  MAX_CHANGES_PER_CALL,
  SEAT_SIZE,
  areAdjacentSeats,
  buildGridLayout,
  buildPendingRow,
  buildSeatChangeBatches,
  chunkLabels,
  cleanSeatLabel,
  diffChangedSeats,
  groupSeatsByRow,
  isPendingSeatId,
  normalizeAisles,
  pendingRowIndexOf,
  rowLabelFromIndex,
  sellableCapacity,
} from '../seatGrid';
import type { ColSpan, Seat, SeatType } from '@/types';

const seat = (
  rowLabel: string,
  colNumber: number,
  extra: Partial<Pick<Seat, 'seat_type' | 'is_gap' | 'col_span'>> = {}
): Seat => ({
  id: `${rowLabel}${colNumber}`,
  hall_id: 'hall-1',
  label: `${rowLabel}${colNumber}`,
  row_label: rowLabel,
  col_number: colNumber,
  seat_type: (extra.seat_type ?? 'standard') as SeatType,
  is_gap: extra.is_gap ?? false,
  col_span: (extra.col_span ?? 1) as ColSpan,
});

describe('groupSeatsByRow', () => {
  it('keeps backend order, never re-sorts alphabetically', () => {
    // Backend sorts by row_index, so row 27 is AA and must stand AFTER Z.
    // String sort would put AA before B - exactly the bug to avoid.
    const rows = groupSeatsByRow([seat('Z', 1), seat('AA', 1), seat('AB', 1)]);
    expect(rows.map((r) => r.rowLabel)).toEqual(['Z', 'AA', 'AB']);
  });

  it('gom dung ghe vao dung hang', () => {
    const rows = groupSeatsByRow([seat('A', 1), seat('A', 2), seat('B', 1)]);
    expect(rows).toHaveLength(2);
    expect(rows[0].seats.map((s) => s.label)).toEqual(['A1', 'A2']);
    expect(rows[1].seats.map((s) => s.label)).toEqual(['B1']);
  });
});

describe('normalizeAisles', () => {
  it('drops out-of-range aisle values', () => {
    // The backend ONLY checks array length (max=49), never values, so real
    // data may hold 0, negatives or over-column numbers.
    expect(normalizeAisles([0, -3, 4, 99, 12], 12)).toEqual([4]);
  });

  it('bo trung lap va sap tang dan', () => {
    expect(normalizeAisles([10, 4, 4], 14)).toEqual([4, 10]);
  });

  it('bo loi di sau cot cuoi cung vi no khong ve ra gi', () => {
    expect(normalizeAisles([12], 12)).toEqual([]);
  });
});

describe('buildGridLayout', () => {
  it('khong co loi di thi so cot chinh la vach luoi', () => {
    const layout = buildGridLayout(5, []);
    expect(layout.templateColumns).toBe(Array(5).fill(`${SEAT_SIZE}px`).join(' '));
    expect(layout.lineOf(1)).toBe(1);
    expect(layout.lineOf(5)).toBe(5);
  });

  it('chen mot track rieng cho moi loi di va day vach ghe sau no', () => {
    const layout = buildGridLayout(6, [3]);
    expect(layout.templateColumns).toBe(
      `${SEAT_SIZE}px ${SEAT_SIZE}px ${SEAT_SIZE}px ${AISLE_WIDTH}px ${SEAT_SIZE}px ${SEAT_SIZE}px ${SEAT_SIZE}px`
    );
    // Cols 1..3 unchanged; col 4+ shifts right one track.
    expect(layout.lineOf(3)).toBe(3);
    expect(layout.lineOf(4)).toBe(5);
    expect(layout.lineOf(6)).toBe(7);
  });

  it('ghe doi keo qua ca track loi di khi loi di roi giua no', () => {
    const layout = buildGridLayout(6, [3]);
    // Couple anchored at col 3 covering 3 and 4 with an aisle between -> 3 tracks.
    expect(layout.spanOf(3, 2)).toBe(3);
    // Uncut couple stays 2 tracks.
    expect(layout.spanOf(1, 2)).toBe(2);
    expect(layout.spanOf(1, 1)).toBe(1);
  });

  it('bo qua loi di khong hop le thay vi ve lech luoi', () => {
    const layout = buildGridLayout(4, [0, 9]);
    expect(layout.aisles).toEqual([]);
    expect(layout.lineOf(4)).toBe(4);
  });
});

describe('sellableCapacity', () => {
  it('khong dem o gap, giong COUNT(*) FILTER (WHERE NOT is_gap) cua backend', () => {
    const seats = [seat('A', 1), seat('A', 2, { is_gap: true }), seat('A', 3)];
    expect(sellableCapacity(seats)).toBe(2);
  });

  it('ghe doi van chi la MOT ghe du chiem hai cot', () => {
    expect(sellableCapacity([seat('A', 1, { col_span: 2 }), seat('A', 3)])).toBe(2);
  });
});

describe('cleanSeatLabel', () => {
  it('trims and uppercases, since the backend uppercases but never trims', () => {
    // A padded label matches no seat, and "no match" is a 400 rolling back the WHOLE batch.
    expect(cleanSeatLabel(' a1 ')).toBe('A1');
  });
});

describe('chunkLabels', () => {
  it('duoi 500 nhan thi mot lo, mot thay doi', () => {
    const batches = chunkLabels(['A1', 'A2']);
    expect(batches).toHaveLength(1);
    expect(batches[0]).toHaveLength(1);
    expect(batches[0][0]).toEqual(['A1', 'A2']);
  });

  it('cat theo tran 500 nhan moi thay doi', () => {
    const labels = Array.from({ length: 501 }, (_, i) => `A${i + 1}`);
    const batches = chunkLabels(labels);
    expect(batches).toHaveLength(1);
    expect(batches[0]).toHaveLength(2);
    expect(batches[0][0]).toHaveLength(500);
    expect(batches[0][1]).toHaveLength(1);
  });

  it('tach sang lo goi moi khi vuot 50 thay doi', () => {
    const labels = Array.from({ length: 500 * MAX_CHANGES_PER_CALL + 1 }, (_, i) => `A${i + 1}`);
    const batches = chunkLabels(labels);
    expect(batches).toHaveLength(2);
    expect(batches[0]).toHaveLength(MAX_CHANGES_PER_CALL);
    expect(batches[1]).toHaveLength(1);
  });
});

describe('rowLabelFromIndex', () => {
  it('khop dung ban dich cua dto.RowLabel (Go): A..Z roi AA, AB...', () => {
    expect(rowLabelFromIndex(1)).toBe('A');
    expect(rowLabelFromIndex(26)).toBe('Z');
    expect(rowLabelFromIndex(27)).toBe('AA');
    expect(rowLabelFromIndex(28)).toBe('AB');
    expect(rowLabelFromIndex(52)).toBe('AZ');
    expect(rowLabelFromIndex(53)).toBe('BA');
  });
});

describe('pending row helpers', () => {
  it('buildPendingRow sinh dung so ghe, cung mot id doc duoc lai qua isPendingSeatId/pendingRowIndexOf', () => {
    const row = buildPendingRow(2, 4, 'G', 'hall-1');
    expect(row).toHaveLength(4);
    expect(row.map((s) => s.label)).toEqual(['G1', 'G2', 'G3', 'G4']);
    row.forEach((seat) => {
      expect(isPendingSeatId(seat.id)).toBe(true);
      expect(pendingRowIndexOf(seat.id)).toBe(2);
      expect(seat.seat_type).toBe('standard');
      expect(seat.is_gap).toBe(false);
    });
  });

  it('id ghe THAT (tu server) khong bi coi la pending', () => {
    expect(isPendingSeatId('a1b2c3')).toBe(false);
  });
});

describe('diffChangedSeats', () => {
  it('bo qua ghe pending - chua len server nen khong the la mot THAY DOI', () => {
    const saved = [seat('A', 1)];
    const draft = [...saved, ...buildPendingRow(0, 1, 'B', 'hall-1')];
    expect(diffChangedSeats(saved, draft)).toEqual([]);
  });

  it('bo qua ghe khong doi gi', () => {
    const saved = [seat('A', 1)];
    expect(diffChangedSeats(saved, saved)).toEqual([]);
  });

  it('gom cac ghe co CUNG patch vao mot nhom, khac patch thi nhom rieng', () => {
    const saved = [seat('A', 1), seat('A', 2), seat('A', 3)];
    const draft = [
      { ...saved[0], seat_type: 'vip' as SeatType },
      { ...saved[1], seat_type: 'vip' as SeatType },
      { ...saved[2], is_gap: true },
    ];
    const groups = diffChangedSeats(saved, draft);
    expect(groups).toHaveLength(2);
    const vipGroup = groups.find((g) => g.seat_type === 'vip');
    expect(vipGroup?.labels.sort()).toEqual(['A1', 'A2']);
    const gapGroup = groups.find((g) => g.is_gap === true);
    expect(gapGroup?.labels).toEqual(['A3']);
  });
});

describe('buildSeatChangeBatches', () => {
  it('mot nhom nho thi ra dung mot SeatChange', () => {
    const batches = buildSeatChangeBatches([{ labels: ['A1', 'A2'], seat_type: 'vip' }]);
    expect(batches).toHaveLength(1);
    expect(batches[0]).toEqual([
      { selector: { labels: ['A1', 'A2'] }, seat_type: 'vip', is_gap: undefined },
    ]);
  });

  it('cat mot nhom vuot 500 nhan thanh nhieu SeatChange', () => {
    const labels = Array.from({ length: 501 }, (_, i) => `A${i + 1}`);
    const batches = buildSeatChangeBatches([{ labels, is_gap: true }]);
    expect(batches).toHaveLength(1);
    expect(batches[0]).toHaveLength(2);
    expect(batches[0][0].selector.labels).toHaveLength(500);
    expect(batches[0][1].selector.labels).toHaveLength(1);
  });

  it('gom nhieu nhom lai roi moi cat theo tran 50 thay doi/lan goi', () => {
    const groups = Array.from({ length: MAX_CHANGES_PER_CALL + 1 }, (_, i) => ({
      labels: [`A${i + 1}`],
      seat_type: 'vip' as SeatType,
    }));
    const batches = buildSeatChangeBatches(groups);
    expect(batches).toHaveLength(2);
    expect(batches[0]).toHaveLength(MAX_CHANGES_PER_CALL);
    expect(batches[1]).toHaveLength(1);
  });
});

describe('areAdjacentSeats', () => {
  it('adjacent same-row singles merge', () => {
    expect(areAdjacentSeats(seat('A', 3), seat('A', 4))).toBe(true);
    // Pick order irrelevant.
    expect(areAdjacentSeats(seat('A', 4), seat('A', 3))).toBe(true);
  });

  it('khac hang thi khong ghep duoc du so cot lien tiep', () => {
    expect(areAdjacentSeats(seat('A', 3), seat('B', 4))).toBe(false);
  });

  it('cach nhau hon 1 cot thi khong ghep duoc', () => {
    expect(areAdjacentSeats(seat('A', 3), seat('A', 5))).toBe(false);
  });

  it('mot trong hai da la ghe doi (col_span=2) thi khong ghep duoc', () => {
    expect(areAdjacentSeats(seat('A', 3, { col_span: 2 }), seat('A', 4))).toBe(false);
  });

  it('mot trong hai la o trong thi khong ghep duoc', () => {
    expect(areAdjacentSeats(seat('A', 3, { is_gap: true }), seat('A', 4))).toBe(false);
  });
});
