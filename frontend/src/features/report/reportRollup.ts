import type { DailyAggregate, DailyBreakdownShowtime } from '@/types';

/** Rolls daily figures up by movie and showtime. */
export interface MovieRollupRow {
  movie: string;
  showtimes: number;
  seatsSold: number;
  capacity: number;
  checkedIn: number;
  revenue: number;
}

/** Flattens one showtime per day, dropping days without a breakdown. */
export const flattenShowtimes = (days: DailyAggregate[]): DailyBreakdownShowtime[] =>
  days.flatMap((day) => day.breakdown?.showtimes ?? []);

/** Groups by movie title (a snapshot, no movie_id). */
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
  // Revenue descending; the first question is always which movie earned most.
  return [...byMovie.values()].sort((a, b) => b.revenue - a.revenue);
};

/** Fills the day from the total, as a percent. */
export const occupancyPercent = (seatsSold: number, capacity: number): number =>
  capacity === 0 ? 0 : Math.round((10000 * seatsSold) / capacity) / 100;

/** A day with no rows is missing data, not a zero. */
export const missingDays = (from: string, to: string, days: DailyAggregate[]): string[] => {
  const have = new Set(days.map((d) => d.report_date));
  const out: string[] = [];
  // Iterate in UTC so the viewer's machine timezone never shifts the day.
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
