import dayjs from 'dayjs';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

// Backend offsets are inconsistent (same instant as +07:00 vs Z), and plain dayjs
// formats in the viewer's zone, so every timestamp is pinned to cinema time first.
dayjs.extend(utc);
dayjs.extend(timezone);

/** Cinema zone; matches backend DATABASE_TIMEZONE. */
export const CINEMA_TZ = 'Asia/Ho_Chi_Minh';

export const DATE_FORMAT = 'DD/MM/YYYY';
export const DATETIME_FORMAT = 'DD/MM/YYYY HH:mm';
/** Backend date format for query params and datetime=2006-01-02 binding. */
export const API_DATE_FORMAT = 'YYYY-MM-DD';

/** Parse an API timestamp into cinema time. Apply before any display formatting. */
export const toCinemaTime = (value: string | number | Date) => dayjs(value).tz(CINEMA_TZ);

export const formatDate = (value?: string | null) =>
  value ? toCinemaTime(value).format(DATE_FORMAT) : '-';

export const formatDateTime = (value?: string | null) =>
  value ? toCinemaTime(value).format(DATETIME_FORMAT) : '-';

/** Format an instant as the API `date` param (YYYY-MM-DD in cinema time). */
export const toApiDate = (value?: string | number | Date | null) =>
  value ? toCinemaTime(value).format(API_DATE_FORMAT) : '';

export const formatDuration = (minutes?: number | null) => {
  if (!minutes || minutes <= 0) return '-';
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
};

// Money is int64 whole VND: no subunits, no currency field.
const vndFormatter = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 0 });

/** 120000 -> "120.000 ₫". Use for every displayed money value. */
export const formatVND = (amount?: number | null) =>
  amount === null || amount === undefined ? '-' : `${vndFormatter.format(amount)} ₫`;

/** 120000 -> "120.000" when the unit label lives elsewhere. */
export const formatNumber = (value?: number | null) =>
  value === null || value === undefined ? '-' : vndFormatter.format(value);

/** Bridge form date-pickers (user-zone) and API instants (cinema-zone). DatePicker works in the user's zone while operators think in cinema time; on other-zone machines the two drift and shows get created at the wrong hour. Read input as cinema time and back. */
export const toApiInstant = (value: dayjs.Dayjs): string =>
  dayjs.tz(value.format('YYYY-MM-DDTHH:mm:ss'), CINEMA_TZ).format();

/** Inverse of toApiInstant: API instant -> cinema time for form display. */
export const fromApiInstant = (value: string): dayjs.Dayjs =>
  dayjs(toCinemaTime(value).format('YYYY-MM-DDTHH:mm:ss'));
