import dayjs from 'dayjs';

export const DATE_FORMAT = 'DD/MM/YYYY';
export const DATETIME_FORMAT = 'DD/MM/YYYY HH:mm';

export const formatDate = (value?: string | null) =>
  value ? dayjs(value).format(DATE_FORMAT) : '-';

export const formatDateTime = (value?: string | null) =>
  value ? dayjs(value).format(DATETIME_FORMAT) : '-';

export const formatDuration = (minutes?: number | null) => {
  if (!minutes || minutes <= 0) return '-';
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
};
