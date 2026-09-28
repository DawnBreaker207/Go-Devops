import dayjs from 'dayjs';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(timezone);

export const CINEMA_TZ = 'Asia/Ho_Chi_Minh';

export const DATE_FORMAT = 'DD/MM/YYYY';
export const DATETIME_FORMAT = 'DD/MM/YYYY HH:mm';
export const API_DATE_FORMAT = 'YYYY-MM-DD';

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

const vndFormatter = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 0 });

export const formatVND = (amount?: number | null) =>
  amount === null || amount === undefined ? '-' : `${vndFormatter.format(amount)} ₫`;

export const formatNumber = (value?: number | null) =>
  value === null || value === undefined ? '-' : vndFormatter.format(value);

export const toApiInstant = (value: dayjs.Dayjs): string =>
  dayjs.tz(value.format('YYYY-MM-DDTHH:mm:ss'), CINEMA_TZ).format();

export const fromApiInstant = (value: string): dayjs.Dayjs =>
  dayjs(toCinemaTime(value).format('YYYY-MM-DDTHH:mm:ss'));
