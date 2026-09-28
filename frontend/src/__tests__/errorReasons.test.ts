import { describe, expect, it } from 'vitest';
import vi from '@/locales/vi.json';
import en from '@/locales/en.json';
import { ERROR_REASONS } from '@/utils/errorReasons';
import { isRecoverableRefreshError, reasonMessage } from '@/utils/error';

type Dict = Record<string, string>;
const viErr = (vi as unknown as { err: Dict }).err;
const enErr = (en as unknown as { err: Dict }).err;

// A missing key never breaks the build: i18next returns the defaultValue (an English sentence) inside Vietnamese UI.
describe('err.<reason> phu kin moi ly do backend co the tra ve', () => {
  it('moi reason deu co ban dich tieng Viet', () => {
    const missing = ERROR_REASONS.filter((r) => !viErr[r]);
    expect(missing, `thieu ban dich vi cho: ${missing.join(', ')}`).toEqual([]);
  });

  it('moi reason deu co ban dich tieng Anh', () => {
    const missing = ERROR_REASONS.filter((r) => !enErr[r]);
    expect(missing, `thieu ban dich en cho: ${missing.join(', ')}`).toEqual([]);
  });

  // A surplus key means the backend renamed or dropped a sentinel this side hasn't cleaned up.
  it('khong co khoa thua trong hai file ngon ngu', () => {
    const known = new Set<string>(ERROR_REASONS);
    expect(Object.keys(viErr).filter((k) => !known.has(k))).toEqual([]);
    expect(Object.keys(enErr).filter((k) => !known.has(k))).toEqual([]);
  });

  it('ban dich tieng Viet khong bo trong va khong phai chinh khoa', () => {
    for (const r of ERROR_REASONS) {
      expect(viErr[r].trim().length, `${r} rong`).toBeGreaterThan(0);
      expect(viErr[r]).not.toBe(r);
    }
  });
});

describe('reasonMessage', () => {
  // A real t() in the app returns defaultValue on a missing key; mock it the same way.
  const t = (key: string, options?: { defaultValue?: string }) => {
    const reason = key.startsWith('err.') ? key.slice(4) : '';
    return viErr[reason] ?? options?.defaultValue ?? key;
  };

  it('dich theo reason chu khong theo cau chu', () => {
    const error = {
      code: 40900,
      message: 'a payment is already in progress for your pending booking',
      reason: 'payment_in_progress',
    };
    expect(reasonMessage(error, t, 'du phong')).toBe(viErr.payment_in_progress);
  });

  // The old table (discountErrors.ts) keyed on sentences, so a backend rewording fell back to English.
  it('doi cau chu backend van dich dung', () => {
    const error = {
      code: 40001,
      message: 'this discount code has expired, sorry!',
      reason: 'discount_expired',
    };
    expect(reasonMessage(error, t, 'du phong')).toBe(viErr.discount_expired);
  });

  it('khong co reason thi dung duong cu (safeMessage)', () => {
    const error = { code: 40001, message: 'mot loi khong quen' };
    expect(reasonMessage(error, t, 'du phong')).toBe('mot loi khong quen');
  });

  it('reason la thu khong biet thi roi ve fallback chu khong in ra khoa', () => {
    const error = { code: 40001, message: 'raw backend sentence', reason: 'khong_ton_tai' };
    expect(reasonMessage(error, t, 'du phong')).toBe('raw backend sentence');
  });

  it('loi khong phai tu API van dung fallback', () => {
    expect(reasonMessage(new Error('boom'), t, 'du phong')).toBe('du phong');
  });

  // safeMessage hides technical detail sentences; reasonMessage must preserve that.
  it('van giau thong diep lo SQLSTATE', () => {
    const error = { code: 50000, message: 'pq: duplicate key value violates constraint' };
    expect(reasonMessage(error, t, 'du phong')).toBe('du phong');
  });
});

describe('isRecoverableRefreshError', () => {
  // 409 payment_in_progress is an open checkout, not a lost order: treating it as
  // failure would make BookingFlowPage cancel the order right before payment lands.
  it('409 payment_in_progress KHONG duoc coi la mat don', () => {
    expect(
      isRecoverableRefreshError({
        code: 40900,
        message: 'a payment is already in progress for your pending booking',
        reason: 'payment_in_progress',
      })
    ).toBe(true);
  });

  it('cac loi khac van la mat don that su', () => {
    expect(isRecoverableRefreshError({ code: 40900, reason: 'booking_expired', message: '' })).toBe(
      false
    );
    expect(isRecoverableRefreshError({ code: 40900, reason: 'seat_taken', message: '' })).toBe(
      false
    );
    expect(
      isRecoverableRefreshError({ code: 40900, reason: 'hold_lifetime_exceeded', message: '' })
    ).toBe(false);
  });

  // Keyed by reason, never by sentence, so a backend rewording can't turn a harmless error into a cancelled order.
  it('chi nhin reason, khong nhin message', () => {
    expect(
      isRecoverableRefreshError({
        code: 40900,
        message: 'a payment is already in progress for your pending booking',
      })
    ).toBe(false);
  });

  it('loi khong phai tu API thi khong phai loai phuc hoi duoc', () => {
    expect(isRecoverableRefreshError(new Error('mat mang'))).toBe(false);
    expect(isRecoverableRefreshError(null)).toBe(false);
  });
});
