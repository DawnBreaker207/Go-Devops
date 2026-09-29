import { describe, expect, it } from 'vitest';
import {
  AISLE_WIDTH,
  MAX_CHANGES_PER_CALL,
  SEAT_SIZE,
  ZOOM_MAX,
  ZOOM_MIN,
  areAdjacentSeats,
  buildGridLayout,
  buildPendingColumn,
  buildPendingRow,
  buildSeatChangeBatches,
  chunkLabels,
  cleanSeatLabel,
  clampZoom,
  computeFitScale,
  diffChangedSeats,
  diffMergeSplitOps,
  gridPixelWidth,
  groupSeatsByRow,
  isPendingColSeatId,
  isPendingSeatId,
  isPendingSplitSeatId,
  isUnsavedSeatId,
  makeSplitSeatId,
  normalizeAisles,
  pendingColIndexOf,
  pendingRowIndexOf,
  rowLabelFromIndex,
  sellableCapacity,
  isPendingSingleSeatId,
  isVirtualGapSeatId,
  makePendingSingleSeatId,
  makeVirtualGapId,
  trimTrailingGapColumns,
  virtualTrailingSlots,
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
  has_booking_history: false,
});

describe('groupSeatsByRow', () => {
  it('keeps backend order, never re-sorts alphabetically', () => {
    // Backend sorts by row_index, so row 27 is AA and must stand after Z; a string sort would wrongly put AA before B.
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
    // Backend only checks array length (max=49), never values, so real data may hold 0/negatives/over-column numbers.
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

  it('buildPendingRow voi totalWidth rong hon seatsPerRow: cot du la O TRONG, khong phai ghe thuong (co pending column tu truoc)', () => {
    const row = buildPendingRow(2, 4, 'G', 'hall-1', 6);
    expect(row).toHaveLength(6);
    expect(row.map((s) => s.is_gap)).toEqual([false, false, false, false, true, true]);
  });
});

describe('pending column helpers', () => {
  it('buildPendingColumn sinh 1 ghe THUONG moi hang, khong phai o trong', () => {
    const col = buildPendingColumn(0, ['A', 'B', 'C'], 5, 'hall-1');
    expect(col).toHaveLength(3);
    expect(col.map((s) => s.label)).toEqual(['A5', 'B5', 'C5']);
    col.forEach((seat) => {
      expect(isPendingColSeatId(seat.id)).toBe(true);
      expect(isUnsavedSeatId(seat.id)).toBe(true);
      expect(pendingColIndexOf(seat.id)).toBe(0);
      expect(seat.seat_type).toBe('standard');
      expect(seat.is_gap).toBe(false);
      expect(seat.col_number).toBe(5);
    });
  });

  it('pending row va pending column dung prefix rieng, khong bi nham lan', () => {
    const rowSeat = buildPendingRow(0, 1, 'A', 'hall-1')[0];
    const colSeat = buildPendingColumn(0, ['A'], 1, 'hall-1')[0];
    expect(isPendingSeatId(rowSeat.id)).toBe(true);
    expect(isPendingColSeatId(rowSeat.id)).toBe(false);
    expect(isPendingSeatId(colSeat.id)).toBe(false);
    expect(isPendingColSeatId(colSeat.id)).toBe(true);
  });

  it('id ghe THAT khong bi coi la unsaved', () => {
    expect(isUnsavedSeatId('a1b2c3')).toBe(false);
  });

  it('ghe le pending duoc nhan dien la unsaved (luu qua POST /seats)', () => {
    const id = makePendingSingleSeatId('B', 7);
    expect(isPendingSingleSeatId(id)).toBe(true);
    expect(isUnsavedSeatId(id)).toBe(true);
    expect(isPendingSingleSeatId('pending-col-0-B')).toBe(false);
  });
});

describe('virtualTrailingSlots', () => {
  it('hang thieu cot moi nhat duoc bu o ao co nut + (khong vao draft)', () => {
    const seats = [
      ...Array.from({ length: 5 }, (_, i) => seat('A', i + 1)),
      ...Array.from({ length: 4 }, (_, i) => seat('B', i + 1)),
    ];
    const virtuals = virtualTrailingSlots(seats, seats);
    expect(virtuals.map((s) => s.label)).toEqual(['B5']);
    expect(virtuals[0]?.is_gap).toBe(true);
    expect(virtuals[0]?.id).toBe(makeVirtualGapId('B', 5));
    expect(isVirtualGapSeatId(virtuals[0]?.id ?? '')).toBe(true);
    expect(isVirtualGapSeatId('pending-single-B-5')).toBe(false);
  });

  it('khong sinh o ao khi cac hang deu du cot, list rong tra rong', () => {
    const seats = [seat('A', 1), seat('B', 1)];
    expect(virtualTrailingSlots(seats, seats)).toEqual([]);
    expect(virtualTrailingSlots([], [])).toEqual([]);
  });

  it('o vua bi xoa thi khong hien + lai (baseline van giu lich su that)', () => {
    const trimmed = [
      ...Array.from({ length: 8 }, (_, i) => seat('A', i + 1)),
      { ...buildPendingColumn(0, ['A'], 9, 'hall-1')[0] },
      ...Array.from({ length: 8 }, (_, i) => seat('B', i + 1)),
    ];
    const snapshot = [
      ...Array.from({ length: 8 }, (_, i) => seat('A', i + 1)),
      ...Array.from({ length: 10 }, (_, i) => seat('B', i + 1)),
    ];
    // B9, B10 vua bi xoa trong session: khong offer lai du A9 pending ton tai.
    // Draft van giu object gap B9, B10 (chua luu) nen draft-check cung chan.
    expect(virtualTrailingSlots(trimmed, trimmed, snapshot)).toEqual([]);
    const withHiddenGaps = [
      ...trimmed,
      { ...seat('B', 9), is_gap: true },
      { ...seat('B', 10), is_gap: true },
    ];
    expect(virtualTrailingSlots(trimmed, withHiddenGaps, snapshot)).toEqual([]);
  });

  it('hang tut sau hon 1 cot thi khong offer (di bang nut + cuoi hang)', () => {
    const seats = [
      ...Array.from({ length: 6 }, (_, i) => seat('A', i + 1)),
      ...Array.from({ length: 4 }, (_, i) => seat('B', i + 1)),
    ];
    // B max 4, grid cuoi cot 6: 4+1 != 6 nen khong co o ao nao.
    expect(virtualTrailingSlots(seats, seats)).toEqual([]);
  });

  it('cot that su moi thi van offer + (baseline chua tung co)', () => {
    const trimmed = [
      ...Array.from({ length: 5 }, (_, i) => seat('A', i + 1)),
      ...Array.from({ length: 4 }, (_, i) => seat('B', i + 1)),
    ];
    const snapshot = trimmed.map((s) => ({ ...s }));
    const virtuals = virtualTrailingSlots(trimmed, trimmed, snapshot);
    expect(virtuals.map((s) => s.label)).toEqual(['B5']);
  });
});

describe('diffChangedSeats', () => {
  it('bo qua ghe pending row - chua len server nen khong the la mot THAY DOI', () => {
    const saved = [seat('A', 1)];
    const draft = [...saved, ...buildPendingRow(0, 1, 'B', 'hall-1')];
    expect(diffChangedSeats(saved, draft)).toEqual([]);
  });

  it('bo qua ghe pending column - chua len server nen khong the la mot THAY DOI', () => {
    const saved = [seat('A', 1)];
    const draft = [...saved, ...buildPendingColumn(0, ['A'], 2, 'hall-1')];
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

describe('pending split helper', () => {
  it('makeSplitSeatId sinh id doc duoc lai qua isPendingSplitSeatId, va no la unsaved', () => {
    const id = makeSplitSeatId('real-seat-id');
    expect(isPendingSplitSeatId(id)).toBe(true);
    expect(isUnsavedSeatId(id)).toBe(true);
    // Not mistaken for a pending row/column id.
    expect(isPendingSeatId(id)).toBe(false);
    expect(isPendingColSeatId(id)).toBe(false);
  });

  it('id ghe THAT khong bi coi la pending split', () => {
    expect(isPendingSplitSeatId('a1b2c3')).toBe(false);
  });
});

describe('diffMergeSplitOps', () => {
  it('phat hien MOT merge tren cap ghe da luu san (merge/split lam ngay tren draft, khong can save truoc)', () => {
    // Bug scenario: an already-saved couple pair gets merged locally while the grid is otherwise
    // clean - draft mirrors exactly what handleMergeCouple produces (left absorbs, right dropped).
    const baseline = [seat('A', 3), seat('A', 4)];
    const draft = [{ ...seat('A', 3), col_span: 2 as ColSpan, seat_type: 'vip' as SeatType }];
    const { merges, splits } = diffMergeSplitOps(baseline, draft);
    expect(splits).toEqual([]);
    expect(merges).toEqual([{ leftLabel: 'A3', rightLabel: 'A4' }]);
  });

  it('phat hien mot split tren ghe doi da luu san', () => {
    const baseline = [seat('A', 3, { col_span: 2 })];
    const draft = [
      { ...seat('A', 3, { col_span: 1 }) },
      { ...seat('A', 4), id: makeSplitSeatId('A3') },
    ];
    const { merges, splits } = diffMergeSplitOps(baseline, draft);
    expect(merges).toEqual([]);
    expect(splits).toEqual([{ label: 'A3' }]);
  });

  it('phat hien merge NGAY trong mot hang MOI them (chua luu) - dung baseline la seat vua tao', () => {
    // Mirrors handleSave: baseline is savedSnapshot + the seats a fresh addRow just created, all
    // still single-width; the draft already shows the local merge the admin made before Save ran.
    const newSeatDefaults = [seat('G', 1), seat('G', 2), seat('G', 3)];
    const draft = [
      { ...seat('G', 1), col_span: 2 as ColSpan, seat_type: 'couple' as SeatType },
      seat('G', 3),
    ];
    const { merges, splits } = diffMergeSplitOps(newSeatDefaults, draft);
    expect(splits).toEqual([]);
    expect(merges).toEqual([{ leftLabel: 'G1', rightLabel: 'G2' }]);
  });

  it('merge roi split lai dung cap do trong mot phien thi trung hoa, khong sinh thao tac nao', () => {
    const baseline = [seat('A', 3), seat('A', 4)];
    // Net result of merge-then-split-back on the very same pair: both labels present, single again.
    const draft = [seat('A', 3), seat('A', 4)];
    expect(diffMergeSplitOps(baseline, draft)).toEqual({ merges: [], splits: [] });
  });

  it('khong doi gi thi khong co merge/split nao', () => {
    const baseline = [seat('A', 1), seat('A', 2, { col_span: 2 })];
    expect(diffMergeSplitOps(baseline, baseline)).toEqual({ merges: [], splits: [] });
  });

  it('ghe doi bi xoa ca hang (khong con trong draft) khong bi hieu nham la split', () => {
    const baseline = [seat('A', 3, { col_span: 2 })];
    const draft: typeof baseline = [];
    expect(diffMergeSplitOps(baseline, draft)).toEqual({ merges: [], splits: [] });
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

describe('gridPixelWidth', () => {
  it('tinh tong be rong tracks cong gap giua chung', () => {
    const layout = buildGridLayout(5, []);
    // 5 seats * 40px + 4 gaps * 4px default.
    expect(gridPixelWidth(layout.templateColumns)).toBe(5 * SEAT_SIZE + 4 * 4);
  });

  it('tinh ca track loi di', () => {
    const layout = buildGridLayout(6, [3]);
    // 6 seats + 1 aisle track = 7 tracks, 6 gaps.
    expect(gridPixelWidth(layout.templateColumns)).toBe(6 * SEAT_SIZE + AISLE_WIDTH + 6 * 4);
  });

  it('gap tuy chinh', () => {
    const layout = buildGridLayout(3, []);
    expect(gridPixelWidth(layout.templateColumns, 0)).toBe(3 * SEAT_SIZE);
  });
});

describe('computeFitScale', () => {
  it('khong xuong duoi 100% (ZOOM_MIN) du luoi rong hon container - van phai cuon ngang', () => {
    expect(computeFitScale(1000, 500)).toBe(ZOOM_MIN);
  });

  it('khong phong to qua 1 (max mac dinh) du container rong hon luoi', () => {
    expect(computeFitScale(500, 1000)).toBe(1);
  });

  it('khong nho hon ZOOM_MIN du luoi rat rong', () => {
    expect(computeFitScale(10_000, 100)).toBe(ZOOM_MIN);
  });

  it('tra ve mac dinh 1 khi thieu kich thuoc that (chua do duoc DOM)', () => {
    expect(computeFitScale(0, 500)).toBe(1);
    expect(computeFitScale(1000, 0)).toBe(1);
  });
});

describe('clampZoom', () => {
  it('gioi han trong [ZOOM_MIN, ZOOM_MAX]', () => {
    expect(clampZoom(0.1)).toBe(ZOOM_MIN);
    expect(clampZoom(5)).toBe(ZOOM_MAX);
    expect(clampZoom(1.2)).toBe(1.2);
  });
});

describe('trimTrailingGapColumns', () => {
  const gap = (row: string, col: number): Seat => ({
    ...seat(row, col),
    is_gap: true,
  });

  it('cot con ghe that thi giu ca o trong cung cot (kem nut + de lap lai)', () => {
    const seats = [
      seat('A', 1),
      seat('A', 2),
      gap('A', 3),
      gap('A', 4),
      seat('B', 1),
      gap('B', 2),
      gap('B', 3),
      gap('B', 4),
    ];
    const baseline = seats.map((s) => ({ ...s }));
    const trimmed = trimTrailingGapColumns(seats, baseline);
    // Cot 2 con ghe that A2 nen B2 van hien; cot 3-4 la dirt cu nen mat.
    expect(trimmed.map((s) => s.label).sort()).toEqual(['A1', 'A2', 'B1', 'B2']);
  });

  it('o vua xoa trong session van hien kem nut + (khong don viec dang lam)', () => {
    const saved = Array.from({ length: 10 }, (_, i) => seat('A', i + 1));
    const draft = [...saved.slice(0, 9), { ...saved[9], is_gap: true }];
    const trimmed = trimTrailingGapColumns(draft, saved);
    expect(trimmed.map((s) => s.label).sort()).toEqual(
      Array.from({ length: 10 }, (_, i) => `A${i + 1}`).sort()
    );
  });

  it('khong cat gi khi cot cuoi co ghe that', () => {
    const seats = [seat('A', 1), gap('A', 2), seat('A', 3)];
    expect(trimTrailingGapColumns(seats)).toHaveLength(3);
  });

  it('ghe pending that vuot maxReal van giu (ghe vua tach, cot moi them)', () => {
    const realPending = { ...buildPendingColumn(0, ['A'], 5, 'hall-1')[0] };
    const seats = [seat('A', 1), realPending, seat('B', 1)];
    const trimmed = trimTrailingGapColumns(seats);
    expect(trimmed.some((s) => s.id === realPending.id)).toBe(true);
  });

  it('duoi persisted don ngay ca khi hang khac co ghe pending moi (maxReal loai pending)', () => {
    const pendingA9 = { ...buildPendingColumn(0, ['A'], 9, 'hall-1')[0] };
    const seats = [
      ...Array.from({ length: 8 }, (_, i) => seat('A', i + 1)),
      pendingA9,
      ...Array.from({ length: 8 }, (_, i) => seat('B', i + 1)),
      { ...seat('B', 9), is_gap: true },
      { ...seat('B', 10), is_gap: true },
    ];
    const baseline = [
      ...Array.from({ length: 8 }, (_, i) => seat('A', i + 1)),
      ...Array.from({ length: 8 }, (_, i) => seat('B', i + 1)),
      { ...seat('B', 9), is_gap: true },
      { ...seat('B', 10), is_gap: true },
    ];
    const trimmed = trimTrailingGapColumns(seats, baseline);
    expect(trimmed.some((s) => s.id === pendingA9.id)).toBe(true);
    expect(trimmed.some((s) => s.label === 'B9')).toBe(false);
    expect(trimmed.some((s) => s.label === 'B10')).toBe(false);
  });

  it('gap pending vuot maxReal van giu (draft dang dung do, chua luu)', () => {
    const gapPending = { ...buildPendingColumn(1, ['B'], 5, 'hall-1')[0], is_gap: true };
    const seats = [seat('A', 1), seat('B', 1), gapPending];
    const trimmed = trimTrailingGapColumns(seats);
    expect(trimmed.some((s) => s.id === gapPending.id)).toBe(true);
  });

  it('hang toan gap thi giu nguyen ca hang (van lap lai duoc)', () => {
    const seats = [
      seat('A', 1),
      { ...seat('B', 1), is_gap: true },
      { ...seat('B', 2), is_gap: true },
    ];
    const trimmed = trimTrailingGapColumns(seats);
    expect(trimmed.map((s) => s.label).sort()).toEqual(['A1', 'B1', 'B2']);
  });

  it('luoi rong toan gap thi giu nguyen (khong trim ve rong)', () => {
    const seats = [gap('A', 1), gap('A', 2)];
    expect(trimTrailingGapColumns(seats)).toHaveLength(2);
  });

  it('case 1 - dirt cu o cot 14: bien mat, con 13 cot', () => {
    const row = Array.from({ length: 14 }, (_, i) => seat('A', i + 1));
    const draft = [...row.slice(0, 13), { ...row[13], is_gap: true }];
    const baseline = draft.map((s) => ({ ...s }));
    const trimmed = trimTrailingGapColumns(draft, baseline);
    expect(trimmed.map((s) => s.col_number)).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13]);
  });

  it('case 2a - cot con ghe that thi o trong cung cot giu lai kem nut +', () => {
    const draft = [
      ...Array.from({ length: 13 }, (_, i) => seat('A', i + 1)),
      { ...seat('A', 14), is_gap: true },
      ...Array.from({ length: 14 }, (_, i) => seat('B', i + 1)),
    ];
    const trimmed = trimTrailingGapColumns(draft);
    // Cot 14 con ghe that B14 nen o trong A14 van hien (de lap lai duoc).
    expect(trimmed.some((s) => s.label === 'A14')).toBe(true);
    expect(trimmed.some((s) => s.label === 'B14')).toBe(true);
    expect(trimmed).toHaveLength(28);
  });

  it('case 2b - dirt cu o cot 14 ca 2 hang: don sach se cot day', () => {
    const draft = [
      ...Array.from({ length: 13 }, (_, i) => seat('A', i + 1)),
      { ...seat('A', 14), is_gap: true },
      ...Array.from({ length: 13 }, (_, i) => seat('B', i + 1)),
      { ...seat('B', 14), is_gap: true },
    ];
    const baseline = draft.map((s) => ({ ...s }));
    const trimmed = trimTrailingGapColumns(draft, baseline);
    expect(trimmed.some((s) => s.col_number === 14)).toBe(false);
    expect(trimmed).toHaveLength(26);
  });

  it('n hang n cot: chi dung o cot dau tien con ghe that', () => {
    const draft = [
      seat('A', 1),
      gap('A', 2),
      gap('A', 3),
      gap('A', 4),
      seat('B', 1),
      seat('B', 2),
      gap('B', 3),
      gap('B', 4),
      seat('C', 1),
      gap('C', 2),
      gap('C', 3),
      gap('C', 4),
    ];
    // Cot 2 con ghe that B2 nen o trong A2, C2 van hien kem nut +;
    // cot 3-4 la dirt cu nen don sach ca 3 hang.
    const baseline = draft.map((s) => ({ ...s }));
    expect(
      trimTrailingGapColumns(draft, baseline)
        .map((s) => s.label)
        .sort()
    ).toEqual(['A1', 'A2', 'B1', 'B2', 'C1', 'C2']);
  });
});
