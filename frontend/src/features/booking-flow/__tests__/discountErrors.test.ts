import { describe, expect, it } from 'vitest';
import { discountErrorMessage } from '../discountErrors';

/** t() stand-in: echoes the key so the mapping itself is what gets asserted. */
const t = ((key: string) => `T:${key}`) as unknown as Parameters<typeof discountErrorMessage>[1];

/** The client normalizes failures into a flat ApiError before they reach a
 *  component, so that is the shape this maps over - not a raw axios error. */
const apiError = (message: string) => ({ code: 40001, message });

describe('discountErrorMessage (map theo CAU CHU, vi moi ly do deu la 40001)', () => {
  it('doi tung ly do sang key rieng', () => {
    expect(discountErrorMessage(apiError('this discount code is not valid'), t, 'fb')).toBe(
      'T:customer.discountInvalid'
    );
    expect(
      discountErrorMessage(apiError('this discount code has been fully redeemed'), t, 'fb')
    ).toBe('T:customer.discountExhausted');
    expect(
      discountErrorMessage(apiError("the order total is below this code's minimum"), t, 'fb')
    ).toBe('T:customer.discountMinOrder');
  });

  it('khong phan biet hoa thuong va khoang trang thua', () => {
    expect(discountErrorMessage(apiError('  THIS DISCOUNT CODE IS NOT VALID  '), t, 'fb')).toBe(
      'T:customer.discountInvalid'
    );
  });

  it('cau la nao thi giu nguyen loi server, khong bia ly do khac', () => {
    expect(discountErrorMessage(apiError('something new from the backend'), t, 'fb')).toBe(
      'something new from the backend'
    );
  });

  it('khong phai loi API thi dung fallback', () => {
    expect(discountErrorMessage(new Error('boom'), t, 'fallback')).toBe('fallback');
  });
});
