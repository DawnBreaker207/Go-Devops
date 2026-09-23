import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  formatDate,
  formatDateTime,
  formatDuration,
  formatNumber,
  formatVND,
  fromApiInstant,
  toApiDate,
  toApiInstant,
  toCinemaTime,
} from '@/utils/format';

describe('format thoi gian', () => {
  // Backend tra cung mot thoi diem duoi hai offset khac nhau:
  // GET /showtimes -> "+07:00", POST /admin/showtimes -> "Z".
  const withOffset = '2026-09-18T17:00:00+07:00';
  const asUtc = '2026-09-18T10:00:00Z';

  it('hien thi cung mot gio rap du offset khac nhau', () => {
    expect(formatDateTime(withOffset)).toBe('18/09/2026 17:00');
    expect(formatDateTime(asUtc)).toBe(formatDateTime(withOffset));
  });

  it('khong troi theo mui gio cua may chay test', () => {
    // Neu format bang dayjs tran, may UTC se ra 10:00 thay vi 17:00.
    expect(toCinemaTime(asUtc).hour()).toBe(17);
  });

  it('formatDate va toApiDate lay ngay theo gio rap', () => {
    expect(formatDate(asUtc)).toBe('18/09/2026');
    expect(toApiDate(asUtc)).toBe('2026-09-18');
  });

  it('gia tri rong tra ve dau gach', () => {
    expect(formatDate(null)).toBe('-');
    expect(formatDateTime(undefined)).toBe('-');
    expect(toApiDate(null)).toBe('');
  });

  it('mot thoi diem sat nua dem UTC van roi dung ngay hom sau o gio rap', () => {
    // 2026-09-18T18:00:00Z = 2026-09-19T01:00:00+07:00
    expect(toApiDate('2026-09-18T18:00:00Z')).toBe('2026-09-19');
  });
});

describe('format tien va thoi luong', () => {
  it('tien la VND nguyen, khong thap phan', () => {
    expect(formatVND(120000)).toBe('120.000 ₫');
    expect(formatVND(0)).toBe('0 ₫');
    expect(formatNumber(70000)).toBe('70.000');
  });

  it('tien rong tra ve dau gach', () => {
    expect(formatVND(null)).toBe('-');
    expect(formatVND(undefined)).toBe('-');
  });

  it('thoi luong doi ra gio va phut', () => {
    expect(formatDuration(128)).toBe('2h 8m');
    expect(formatDuration(45)).toBe('45m');
    expect(formatDuration(0)).toBe('-');
    expect(formatDuration(null)).toBe('-');
  });
});

describe('toApiInstant / fromApiInstant', () => {
  it('form-typed hours go out as cinema time, not machine time', () => {
    // Machine-timezone independent: 19:30 typed in a form becomes 19:30+07:00 on UTC machines too.
    const picked = dayjs('2026-09-22T19:30:00');
    expect(toApiInstant(picked)).toBe('2026-09-22T19:30:00+07:00');
  });

  it('doc nguoc lai ra dung gio rap', () => {
    expect(fromApiInstant('2026-09-22T19:30:00+07:00').format('DD/MM/YYYY HH:mm')).toBe(
      '22/09/2026 19:30'
    );
  });

  it('cung mot thoi diem gui bang Z hay +07:00 deu ra cung gio rap', () => {
    // 12:30Z va 19:30+07:00 la CUNG mot thoi diem - backend tra ca hai kieu.
    const zulu = fromApiInstant('2026-09-22T12:30:00Z');
    const offset = fromApiInstant('2026-09-22T19:30:00+07:00');
    expect(zulu.format()).toBe(offset.format());
    expect(zulu.format('HH:mm')).toBe('19:30');
  });

  it('di duoc ca vong tron form -> API -> form', () => {
    const original = '2026-12-31T23:45:00+07:00';
    expect(toApiInstant(fromApiInstant(original))).toBe(original);
  });
});
