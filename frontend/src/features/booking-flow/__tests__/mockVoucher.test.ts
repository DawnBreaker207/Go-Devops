import { describe, expect, it } from 'vitest';
import { calcMockVoucher, MOCK_VOUCHERS } from '../mockVoucher';

describe('mock voucher (hien thi, khong cham BE)', () => {
  it('co dung 2 ma demo', () => {
    expect(MOCK_VOUCHERS.map((v) => v.code)).toEqual(['WELCOME10', 'GIAM15']);
  });

  it('WELCOME10 tru co dinh 10k', () => {
    expect(calcMockVoucher(140000, 'WELCOME10')).toEqual({ code: 'WELCOME10', discount: 10000 });
  });

  it('khong phan biet hoa thuong va khoang trang', () => {
    expect(calcMockVoucher(140000, '  welcome10 ')).toEqual({ code: 'WELCOME10', discount: 10000 });
  });

  it('GIAM15 tru 15%, tran 50k', () => {
    expect(calcMockVoucher(200000, 'GIAM15')).toEqual({ code: 'GIAM15', discount: 30000 });
    expect(calcMockVoucher(500000, 'GIAM15')).toEqual({ code: 'GIAM15', discount: 50000 });
  });

  it('khong giam qua tam tinh', () => {
    expect(calcMockVoucher(5000, 'WELCOME10')).toEqual({ code: 'WELCOME10', discount: 5000 });
  });

  it('ma la / rong / tam tinh 0 thi null', () => {
    expect(calcMockVoucher(140000, 'KHONGCO')).toBeNull();
    expect(calcMockVoucher(140000, '   ')).toBeNull();
    expect(calcMockVoucher(0, 'WELCOME10')).toBeNull();
  });
});
