/** Display-only mock vouchers (no backend call, real tender untouched). Discounts preview before paying. */
export interface MockVoucherDef {
  code: string;
  kind: 'fixed' | 'percent';
  /** VND for fixed, % for percent. */
  value: number;
  /** Cap (VND), percent only. */
  maxDiscount?: number;
  /** Tagline beside the suggestion chip. */
  hint: string;
}

export const MOCK_VOUCHERS: MockVoucherDef[] = [
  { code: 'WELCOME10', kind: 'fixed', value: 10000, hint: '-10.000đ' },
  { code: 'GIAM15', kind: 'percent', value: 15, maxDiscount: 50000, hint: '-15% tối đa 50k' },
];

export interface MockDiscount {
  code: string;
  discount: number;
}

/** Discount in VND (rounded down) - null for blank/unknown codes. */
export const calcMockVoucher = (subtotal: number, rawCode: string): MockDiscount | null => {
  const code = rawCode.trim().toUpperCase();
  if (code.length === 0) return null;
  const def = MOCK_VOUCHERS.find((v) => v.code === code);
  if (!def || subtotal <= 0) return null;
  const raw = def.kind === 'fixed' ? def.value : Math.floor((subtotal * def.value) / 100);
  const discount = Math.min(raw, def.maxDiscount ?? raw, subtotal);
  if (discount <= 0) return null;
  return { code: def.code, discount };
};
