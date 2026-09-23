/** Shared admin-tab date ranges: period presets, buckets, gap-fill, trend. */

/** Period presets: current week / month / quarter / year. */
export type DashboardPreset = 'week' | 'month' | 'quarter' | 'year';

/** Chart bucket per preset (long ranges aggregate). */
export type DashboardBucket = 'day' | 'week' | 'month';

export interface RangeWindow {
  from: string;
  to: string;
}

const iso = (d: Date): string =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

const shift = (base: Date, days: number): Date => {
  const d = new Date(base);
  d.setDate(d.getDate() + days);
  return d;
};

const startOfWeek = (d: Date): Date => {
  const out = new Date(d);
  const dow = (out.getDay() + 6) % 7; // Monday-first, like CinePlex week view
  out.setDate(out.getDate() - dow);
  out.setHours(0, 0, 0, 0);
  return out;
};

const startOfMonth = (d: Date): Date => new Date(d.getFullYear(), d.getMonth(), 1);

const startOfYear = (d: Date): Date => new Date(d.getFullYear(), 0, 1);

const monthEnd = (start: Date): Date => new Date(start.getFullYear(), start.getMonth() + 1, 0);

/** Period type + anchor pick (any week/month/year date; explicit Q+year). */
export type FilterSelection =
  | { type: 'week' | 'month' | 'year'; date: Date }
  | { type: 'quarter'; quarter: number; year: number };

const FILTER_BUCKET: Record<DashboardPreset, DashboardBucket> = {
  week: 'day',
  month: 'day',
  quarter: 'week',
  year: 'month',
};

export interface PresetWindows {
  current: RangeWindow;
  previous: RangeWindow;
  bucket: DashboardBucket;
  /** Days in the current window (for "vs prior Nd" labels). */
  span: number;
}

/** Full anchored period (Mon-Sun week, full month/quarter/year). Unclamped: future days chart empty. */
export const filterWindows = (sel: FilterSelection): PresetWindows => {
  let start: Date;
  let end: Date;
  if (sel.type === 'quarter') {
    const m = (sel.quarter - 1) * 3;
    start = new Date(sel.year, m, 1);
    end = new Date(sel.year, m + 3, 0);
  } else {
    const d = new Date(sel.date.getFullYear(), sel.date.getMonth(), sel.date.getDate());
    if (sel.type === 'week') {
      start = startOfWeek(d);
      end = shift(start, 6);
    } else if (sel.type === 'month') {
      start = startOfMonth(d);
      end = monthEnd(start);
    } else {
      start = startOfYear(d);
      end = new Date(start.getFullYear(), 11, 31);
    }
  }
  const span = Math.round((end.getTime() - start.getTime()) / 86_400_000) + 1;
  return {
    current: { from: iso(start), to: iso(end) },
    previous: { from: iso(shift(start, -span)), to: iso(shift(start, -1)) },
    bucket: FILTER_BUCKET[sel.type],
    span,
  };
};

/** Current-period windows (default anchor: today). */
export const presetWindows = (preset: DashboardPreset, now = new Date()): PresetWindows =>
  preset === 'quarter'
    ? filterWindows({
        type: 'quarter',
        quarter: Math.floor(now.getMonth() / 3) + 1,
        year: now.getFullYear(),
      })
    : filterWindows({ type: preset, date: now });

export interface BucketPoint {
  /** Axis label: MM-DD (day/week-start), MM/YY (month). */
  label: string;
  fullLabel: string;
  revenue: number;
  tickets: number;
}

/** Group filled daily rows into week/month buckets. */
export const bucketize = (
  days: { date: string; revenue: number; tickets: number }[],
  bucket: DashboardBucket
): BucketPoint[] => {
  if (bucket === 'day') {
    return days.map((d) => ({
      label: d.date.slice(5),
      fullLabel: d.date,
      revenue: d.revenue,
      tickets: d.tickets,
    }));
  }
  const groups = new Map<string, { revenue: number; tickets: number }>();
  for (const d of days) {
    const dt = new Date(`${d.date}T00:00:00`);
    const key =
      bucket === 'week'
        ? iso(startOfWeek(dt))
        : `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, '0')}`;
    const g = groups.get(key) ?? { revenue: 0, tickets: 0 };
    g.revenue += d.revenue;
    g.tickets += d.tickets;
    groups.set(key, g);
  }
  return [...groups.entries()].map(([key, g]) => ({
    label: bucket === 'week' ? key.slice(5) : `${key.slice(5)}/${key.slice(2, 4)}`,
    fullLabel: bucket === 'week' ? `w/c ${key}` : key,
    revenue: g.revenue,
    tickets: g.tickets,
  }));
};

/** Fill date gaps with zeros (absence is not data). */
export const fillDays = <T extends { date: string }>(
  from: string,
  to: string,
  rows: T[],
  zero: (date: string) => T
): T[] => {
  const byDate = new Map(rows.map((r) => [r.date, r]));
  const out: T[] = [];
  const start = new Date(`${from}T00:00:00`);
  const end = new Date(`${to}T00:00:00`);
  for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
    const key = iso(d);
    out.push(byDate.get(key) ?? zero(key));
  }
  return out;
};

/** % change vs previous (null on zero base). */
export const trendOf = (current: number, previous: number): number | null =>
  previous === 0 ? null : Math.round(((current - previous) / previous) * 100);
