import { describe, expect, it } from 'vitest';
import { flattenShowtimes, missingDays, occupancyPercent, rollupByMovie } from '../reportRollup';
import type { DailyAggregate, DailyBreakdownShowtime } from '@/types';

const show = (
  movie: string,
  seatsSold: number,
  capacity: number,
  revenue: number,
  extra: Partial<DailyBreakdownShowtime> = {}
): DailyBreakdownShowtime => ({
  showtime_id: `${movie}-${seatsSold}-${capacity}-${revenue}`,
  movie,
  hall: 'Hall 1',
  start_at: '2026-09-18T17:00:00+07:00',
  capacity,
  seats_sold: seatsSold,
  checked_in: 0,
  revenue,
  ...extra,
});

const day = (date: string, showtimes: DailyBreakdownShowtime[] | undefined): DailyAggregate => ({
  report_date: date,
  total_revenue: 0,
  tickets_sold: 0,
  seats_sold: 0,
  capacity: 0,
  occupancy_rate: 0,
  breakdown: showtimes === undefined ? {} : { showtimes },
  updated_at: '2026-09-18T23:59:00+07:00',
});

describe('flattenShowtimes', () => {
  it('chiu duoc ngay khong co khoa showtimes trong breakdown', () => {
    // `breakdown` la jsonb tu do; khong co gi trong schema ep no phai co khoa do.
    const days = [day('2026-09-17', undefined), day('2026-09-18', [show('A', 1, 10, 100)])];
    expect(flattenShowtimes(days)).toHaveLength(1);
  });

  it('gop suat chieu cua moi ngay lai', () => {
    const days = [
      day('2026-09-17', [show('A', 1, 10, 100)]),
      day('2026-09-18', [show('B', 2, 10, 200)]),
    ];
    expect(flattenShowtimes(days).map((s) => s.movie)).toEqual(['A', 'B']);
  });
});

describe('rollupByMovie', () => {
  it('cong don nhieu suat cua cung mot phim, ke ca khac ngay', () => {
    const days = [
      day('2026-09-17', [show('Dune', 10, 60, 700_000), show('Up', 5, 60, 350_000)]),
      day('2026-09-18', [show('Dune', 20, 60, 1_400_000)]),
    ];
    const rows = rollupByMovie(days);
    const dune = rows.find((r) => r.movie === 'Dune');
    expect(dune).toEqual({
      movie: 'Dune',
      showtimes: 2,
      seatsSold: 30,
      capacity: 120,
      checkedIn: 0,
      revenue: 2_100_000,
    });
  });

  it('sap theo doanh thu giam dan, khong theo alphabet', () => {
    const days = [day('2026-09-18', [show('Avatar', 1, 10, 100), show('Barbie', 9, 10, 900)])];
    expect(rollupByMovie(days).map((r) => r.movie)).toEqual(['Barbie', 'Avatar']);
  });

  it('khong co ngay nao thi tra ve mang rong, khong phai loi', () => {
    expect(rollupByMovie([])).toEqual([]);
  });
});

describe('occupancyPercent', () => {
  it('tra ve PHAN TRAM, cung don vi voi occupancy_rate cua backend', () => {
    // Backend lam ROUND(100.0 * seats_sold / capacity), vi du 4/120 -> 3.33.
    expect(occupancyPercent(4, 120)).toBe(3.33);
    expect(occupancyPercent(1, 2)).toBe(50);
  });

  it('suc chua 0 thi tra 0 chu khong chia cho 0', () => {
    expect(occupancyPercent(0, 0)).toBe(0);
  });

  it('cong don truoc roi moi chia, khong lay trung binh cong cua tung suat', () => {
    // Mot suat 1/10 va mot suat 100/200: trung binh cong la 30%, nhung ty le
    // that la 101/210 = 48.1%. Suat to phai nang hon suat nho.
    expect(occupancyPercent(101, 210)).toBe(48.1);
  });
});

describe('missingDays', () => {
  it('liet ke dung nhung ngay backend khong tra ve dong nao', () => {
    const days = [day('2026-09-16', []), day('2026-09-18', [])];
    expect(missingDays('2026-09-15', '2026-09-18', days)).toEqual(['2026-09-15', '2026-09-17']);
  });

  it('ngay co dong nhung toan so 0 KHONG bi coi la thieu', () => {
    // Day la khac biet quan trong nhat cua man hinh: ngay khong ban duoc gi van
    // duoc chot so va van co dong; chi ngay chua chot moi la thieu du lieu.
    const days = [day('2026-09-18', [])];
    expect(missingDays('2026-09-18', '2026-09-18', days)).toEqual([]);
  });

  it('du ngay thi tra ve mang rong', () => {
    const days = [day('2026-09-17', []), day('2026-09-18', [])];
    expect(missingDays('2026-09-17', '2026-09-18', days)).toEqual([]);
  });

  it('chay qua ranh gioi thang ma khong nhay ngay', () => {
    expect(missingDays('2026-08-30', '2026-09-02', [])).toEqual([
      '2026-08-30',
      '2026-08-31',
      '2026-09-01',
      '2026-09-02',
    ]);
  });
});
