import { describe, expect, it } from 'vitest';
import {
  bucketize,
  fillDays,
  filterWindows,
  presetWindows,
} from '@/features/dashboard/components/dashboardRange';

describe('presetWindows', () => {
  // Monday 2026-09-21: week preset covers the full Mon-Sun week.
  const monday = new Date(2026, 8, 21, 12, 0, 0);

  it('week starts Monday, full 7 days unclamped', () => {
    const w = presetWindows('week', monday);
    expect(w.current).toEqual({ from: '2026-09-21', to: '2026-09-27' });
    expect(w.previous).toEqual({ from: '2026-09-14', to: '2026-09-20' });
    expect(w.bucket).toBe('day');
    expect(w.span).toBe(7);
  });

  it('month covers the whole picked month', () => {
    const w = presetWindows('month', new Date(2026, 8, 21, 12, 0, 0));
    expect(w.current).toEqual({ from: '2026-09-01', to: '2026-09-30' });
    expect(w.previous).toEqual({ from: '2026-08-02', to: '2026-08-31' });
    expect(w.span).toBe(30);
  });

  it('quarter buckets by week, year by month', () => {
    expect(presetWindows('quarter', monday).bucket).toBe('week');
    expect(presetWindows('year', monday).bucket).toBe('month');
  });
});

describe('filterWindows', () => {
  it('week containing a Sunday starts the prior Monday', () => {
    // Sunday 2026-09-27 -> Mon 21..Sun 27 (CinePlex Monday-first rule).
    const w = filterWindows({ type: 'week', date: new Date(2026, 8, 27, 9, 0, 0) });
    expect(w.current).toEqual({ from: '2026-09-21', to: '2026-09-27' });
  });

  it('explicit quarter + year', () => {
    const w = filterWindows({ type: 'quarter', quarter: 1, year: 2026 });
    expect(w.current).toEqual({ from: '2026-01-01', to: '2026-03-31' });
    expect(w.span).toBe(90);
  });

  it('year spans Jan-Dec', () => {
    const w = filterWindows({ type: 'year', date: new Date(2026, 5, 15) });
    expect(w.current).toEqual({ from: '2026-01-01', to: '2026-12-31' });
    expect(w.span).toBe(365);
  });
});

describe('fillDays + bucketize', () => {
  it('fills gaps with zeros then groups weeks', () => {
    const filled = fillDays(
      '2026-09-21',
      '2026-09-28',
      [{ date: '2026-09-23', revenue: 100, tickets: 1 }],
      (date) => ({
        date,
        revenue: 0,
        tickets: 0,
      })
    );
    expect(filled).toHaveLength(8);
    const weeks = bucketize(filled, 'week');
    // Mon 21..Sun 27 in one bucket, Mon 28 alone.
    expect(weeks).toHaveLength(2);
    expect(weeks[0].revenue).toBe(100);
    expect(weeks[1]).toMatchObject({ label: '09-28', revenue: 0 });
  });

  it('groups months with numeric labels', () => {
    const months = bucketize(
      [
        { date: '2026-08-30', revenue: 10, tickets: 1 },
        { date: '2026-09-01', revenue: 20, tickets: 2 },
      ],
      'month'
    );
    expect(months).toEqual([
      { label: '08/26', fullLabel: '2026-08', revenue: 10, tickets: 1 },
      { label: '09/26', fullLabel: '2026-09', revenue: 20, tickets: 2 },
    ]);
  });
});
