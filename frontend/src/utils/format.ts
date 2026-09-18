import dayjs from 'dayjs';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

// Backend tra ve RFC3339 nhung offset KHONG dong nhat: GET /showtimes tra +07:00,
// POST /admin/showtimes tra cung thoi diem duoi dang Z. Neu format bang dayjs tran
// thi gio hien thi theo may nguoi xem (may UTC se thay suat 19:00 thanh 12:00).
// Vi vay moi thoi diem deu duoc quy ve gio rap truoc khi format.
dayjs.extend(utc);
dayjs.extend(timezone);

/** Mui gio cua rap, trung voi DATABASE_TIMEZONE cua backend. */
export const CINEMA_TZ = 'Asia/Ho_Chi_Minh';

export const DATE_FORMAT = 'DD/MM/YYYY';
export const DATETIME_FORMAT = 'DD/MM/YYYY HH:mm';
/** Dinh dang ngay ma backend nhan qua query param va binding datetime=2006-01-02. */
export const API_DATE_FORMAT = 'YYYY-MM-DD';

/** Doc mot thoi diem tu API ve gio rap. Dung cho moi timestamp truoc khi hien thi. */
export const toCinemaTime = (value: string | number | Date) => dayjs(value).tz(CINEMA_TZ);

export const formatDate = (value?: string | null) =>
  value ? toCinemaTime(value).format(DATE_FORMAT) : '-';

export const formatDateTime = (value?: string | null) =>
  value ? toCinemaTime(value).format(DATETIME_FORMAT) : '-';

/** Chuyen mot thoi diem thanh tham so `date` cho API (YYYY-MM-DD theo gio rap). */
export const toApiDate = (value?: string | number | Date | null) =>
  value ? toCinemaTime(value).format(API_DATE_FORMAT) : '';

export const formatDuration = (minutes?: number | null) => {
  if (!minutes || minutes <= 0) return '-';
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
};

// Tien la int64 VND NGUYEN - khong co don vi phu, khong co field currency.
const vndFormatter = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 0 });

/** 120000 -> "120.000 ₫". Dung cho moi so tien hien thi tren UI. */
export const formatVND = (amount?: number | null) =>
  amount === null || amount === undefined ? '-' : `${vndFormatter.format(amount)} ₫`;

/** 120000 -> "120.000" khi da co nhan don vi o cho khac. */
export const formatNumber = (value?: number | null) =>
  value === null || value === undefined ? '-' : vndFormatter.format(value);

/**
 * Chuyen doi giua o chon ngay gio cua form va thoi diem gui len API.
 *
 * DatePicker lam viec theo gio CUA MAY nguoi dung, con nguoi van hanh thi nghi
 * theo gio RAP. Tren may dat mui gio khac +07:00, hai thu nay lech nhau va suat
 * chieu se bi tao sai gio. Hai ham duoi day doc gio nguoi dung go duoc nhu la
 * gio rap, va nguoc lai.
 */
export const toApiInstant = (value: dayjs.Dayjs): string =>
  dayjs.tz(value.format('YYYY-MM-DDTHH:mm:ss'), CINEMA_TZ).format();

/** Nghich dao cua toApiInstant: thoi diem tu API -> gio rap de hien trong form. */
export const fromApiInstant = (value: string): dayjs.Dayjs =>
  dayjs(toCinemaTime(value).format('YYYY-MM-DDTHH:mm:ss'));
