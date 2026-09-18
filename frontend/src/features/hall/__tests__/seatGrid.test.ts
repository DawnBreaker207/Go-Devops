import { describe, expect, it } from 'vitest';
import {
  AISLE_WIDTH,
  MAX_CHANGES_PER_CALL,
  SEAT_SIZE,
  buildGridLayout,
  chunkLabels,
  cleanSeatLabel,
  groupSeatsByRow,
  normalizeAisles,
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
  it('giu nguyen thu tu backend tra ve, khong sap lai theo alphabet', () => {
    // Backend sap theo row_index, nen hang thu 27 la AA va phai dung SAU Z.
    // Sap chuoi se dua AA len truoc B - day chinh la loi phai tranh.
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
  it('bo gia tri ngoai khoang ve duoc', () => {
    // Backend CHI kiem tra do dai mang (max=49), khong kiem tra gia tri, nen
    // du lieu that co the chua 0, so am hoac so vuot so cot.
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
    // Cot 1..3 khong doi; cot 4 tro di bi day sang phai mot track.
    expect(layout.lineOf(3)).toBe(3);
    expect(layout.lineOf(4)).toBe(5);
    expect(layout.lineOf(6)).toBe(7);
  });

  it('ghe doi keo qua ca track loi di khi loi di roi giua no', () => {
    const layout = buildGridLayout(6, [3]);
    // Ghe doi neo o cot 3 phu cot 3 va 4, ma loi di lai nam giua -> 3 track.
    expect(layout.spanOf(3, 2)).toBe(3);
    // Ghe doi khong bi loi di cat thi van la 2 track.
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
  it('trim va viet hoa, vi backend chi viet hoa chu khong trim', () => {
    // Mot nhan thua khoang trang khong khop ghe nao, va "khong khop ghe nao"
    // la loi 400 lam rollback CA LO thay doi.
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
