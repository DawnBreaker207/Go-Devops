/// <reference types="node" />
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { brand, cinemaBackdrop, seat, seatType, semantic, textOnBrand } from '@/theme/tokens';

// Doc thang file CSS tu dia. Khong dung `@/index.css?raw` vi vite.config.ts dat
// test.css = false, nen moi import CSS (ke ca ?raw) deu bi stub thanh rong.
const css = readFileSync(resolve(process.cwd(), 'src/index.css'), 'utf8');

const cssVar = (name: string): string => {
  const match = css.match(new RegExp(`--cp-${name}:\\s*([^;]+);`));
  if (!match) throw new Error(`thieu bien CSS --cp-${name} trong src/index.css`);
  return match[1].trim().toLowerCase();
};

/**
 * tokens.ts la nguon su that; index.css chi la ban sao cho phia CSS. Test nay ton
 * tai de mot ben doi ma ben kia quen doi thi build do ngay, thay vi de hai he mau
 * troi xa nhau am tham.
 */
describe('design token: TS va CSS phai trung nhau', () => {
  const pairs: Array<[string, string]> = [
    ['brand', brand.base],
    ['brand-hover', brand.hover],
    ['brand-active', brand.active],
    ['brand-soft', brand.soft],
    ['brand-softer', brand.softer],
    ['text-on-brand', textOnBrand],
    ['backdrop-base', cinemaBackdrop.base],
    ['backdrop-mid', cinemaBackdrop.mid],
    ['backdrop-glow', cinemaBackdrop.glow],
    ['danger', semantic.danger],
    ['warning', semantic.warning],
    ['info', semantic.info],
    ['seat-available', seat.available],
    ['seat-available-text', seat.availableText],
    ['seat-selected', seat.selected],
    ['seat-selected-text', seat.selectedText],
    ['seat-sold', seat.sold],
    ['seat-sold-text', seat.soldText],
    ['seat-held', seat.held],
    ['seat-held-text', seat.heldText],
    ['seat-screen', seat.screen],
    ['seat-type-standard', seatType.standard.bg],
    ['seat-type-standard-text', seatType.standard.fg],
    ['seat-type-vip', seatType.vip.bg],
    ['seat-type-vip-text', seatType.vip.fg],
    ['seat-type-couple', seatType.couple.bg],
    ['seat-type-couple-text', seatType.couple.fg],
    ['seat-type-recliner', seatType.recliner.bg],
    ['seat-type-recliner-text', seatType.recliner.fg],
  ];

  it.each(pairs)('--cp-%s khop tokens.ts', (name, value) => {
    expect(cssVar(name)).toBe(value.toLowerCase());
  });
});

const luminance = (hex: string) => {
  const n = Number.parseInt(hex.slice(1), 16);
  const channel = (c: number) => {
    const v = c / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  };
  return (
    0.2126 * channel((n >> 16) & 255) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255)
  );
};

const contrast = (a: string, b: string) => {
  const la = luminance(a);
  const lb = luminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
};

describe('design token: cac rang buoc phai giu', () => {
  it('mau thanh cong dung chung voi brand, dung mot mau thu hai', () => {
    expect(semantic.success).toBe(brand.base);
  });

  it('ghe dang chon dung dung mau brand', () => {
    expect(seat.selected).toBe(brand.base);
  });

  it('chu tren nen brand du tuong phan de doc', () => {
    // Figma de chu trang tren #1DE782 (~1.4:1). Test nay chan viec quay lai do.
    expect(contrast(brand.base, textOnBrand)).toBeGreaterThanOrEqual(4.5);
  });

  it.each(Object.entries(seatType))('nhan ghe %s doc duoc tren nen cua no', (_name, pair) => {
    expect(contrast(pair.bg, pair.fg)).toBeGreaterThanOrEqual(4.5);
  });

  it('bon loai ghe co bon mau nen khac nhau', () => {
    const backgrounds = Object.values(seatType).map((pair) => pair.bg);
    expect(new Set(backgrounds).size).toBe(backgrounds.length);
  });
});
