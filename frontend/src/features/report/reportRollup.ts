import type { DailyAggregate, DailyBreakdownShowtime } from '@/types';

/**
 * Gop du lieu bao cao lai theo phim va theo suat chieu.
 *
 * Backend KHONG co endpoint nao tra ve doanh thu theo phim - no chi chot so
 * theo NGAY, va nhet danh sach suat chieu cua ngay do vao cot jsonb
 * `breakdown.showtimes`. Bang "theo phim" cua thiet ke vi vay phai duoc gop o
 * day. Tach khoi React de test duoc, vi day la cho de sai nhat cua man hinh.
 */

export interface MovieRollupRow {
  movie: string;
  showtimes: number;
  seatsSold: number;
  capacity: number;
  checkedIn: number;
  revenue: number;
}

/** Lay phang moi suat chieu cua moi ngay, bo qua ngay khong co breakdown. */
export const flattenShowtimes = (days: DailyAggregate[]): DailyBreakdownShowtime[] =>
  days.flatMap((day) => day.breakdown?.showtimes ?? []);

/**
 * Gop theo TEN phim chu khong theo id: breakdown chi chup lai ten, khong luu
 * movie_id. Hai phim trung ten se bi gop lam mot - chap nhan duoc, vi ten phim
 * co rang buoc duy nhat o tang catalogue va bao cao la anh chup qua khu.
 */
export const rollupByMovie = (days: DailyAggregate[]): MovieRollupRow[] => {
  const byMovie = new Map<string, MovieRollupRow>();
  flattenShowtimes(days).forEach((show) => {
    const row = byMovie.get(show.movie) ?? {
      movie: show.movie,
      showtimes: 0,
      seatsSold: 0,
      capacity: 0,
      checkedIn: 0,
      revenue: 0,
    };
    row.showtimes += 1;
    row.seatsSold += show.seats_sold;
    row.capacity += show.capacity;
    row.checkedIn += show.checked_in;
    row.revenue += show.revenue;
    byMovie.set(show.movie, row);
  });
  // Doanh thu giam dan: cau hoi dau tien cua nguoi xem bao cao la "phim nao thu
  // duoc nhieu nhat", khong phai thu tu alphabet.
  return [...byMovie.values()].sort((a, b) => b.revenue - a.revenue);
};

/**
 * Do lap day tinh tu hai so da gop, KHONG phai trung binh cong cua tung
 * `occupancy_rate`: trung binh cong coi mot suat 10 ghe va mot suat 200 ghe
 * nang bang nhau. Tra ve phan tram, cung don vi voi `occupancy_rate` cua backend.
 */
export const occupancyPercent = (seatsSold: number, capacity: number): number =>
  capacity === 0 ? 0 : Math.round((10000 * seatsSold) / capacity) / 100;

/**
 * Nhung ngay trong khoang ma backend KHONG tra ve dong nao.
 *
 * Day khong phai ngay khong ban duoc gi - ngay do van co dong voi toan so 0.
 * Day la nhung ngay job `closeDay` chua chay, tuc so lieu con THIEU. Phai noi
 * ro ra, khong duoc coi nhu 0.
 */
export const missingDays = (from: string, to: string, days: DailyAggregate[]): string[] => {
  const have = new Set(days.map((d) => d.report_date));
  const out: string[] = [];
  // Duyet bang UTC de khong bi mui gio cua may nguoi xem lam lech mot ngay:
  // `from`/`to` la chuoi YYYY-MM-DD cua gio rap, khong phai mot thoi diem.
  const cursor = new Date(`${from}T00:00:00Z`);
  const end = new Date(`${to}T00:00:00Z`);
  if (Number.isNaN(cursor.getTime()) || Number.isNaN(end.getTime())) return out;
  while (cursor <= end) {
    const key = cursor.toISOString().slice(0, 10);
    if (!have.has(key)) out.push(key);
    cursor.setUTCDate(cursor.getUTCDate() + 1);
  }
  return out;
};
