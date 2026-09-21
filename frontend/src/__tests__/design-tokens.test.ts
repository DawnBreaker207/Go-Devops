/// <reference types="node" />
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import {
  brand,
  brandAlpha,
  cinemaBackdrop,
  customerLightCanvas,
  ratingColor,
  seat,
  seatType,
  semantic,
  surface,
  textOnBrand,
} from '@/theme/tokens';

// Read the CSS file straight from disk. No `@/index.css?raw`: vite.config sets
// test.css = false, so every CSS import (even ?raw) is stubbed empty.
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
    ['light-canvas', customerLightCanvas],
    ['danger', semantic.danger],
    ['warning', semantic.warning],
    ['info', semantic.info],
    ['seat-available', seat.available],
    ['seat-available-text', seat.availableText],
    ['seat-available-hover', seat.availableHover],
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
    ['surface-base', surface.light.base],
    ['surface-border', surface.light.border],
    ['rating-p', ratingColor.p.bg],
    ['rating-p-text', ratingColor.p.fg],
    ['rating-k', ratingColor.k.bg],
    ['rating-k-text', ratingColor.k.fg],
    ['rating-t13', ratingColor.t13.bg],
    ['rating-t13-text', ratingColor.t13.fg],
    ['rating-t16', ratingColor.t16.bg],
    ['rating-t16-text', ratingColor.t16.fg],
    ['rating-t18', ratingColor.t18.bg],
    ['rating-t18-text', ratingColor.t18.fg],
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

describe('design tokens: invariants to keep', () => {
  it('success color differs from brand and danger', () => {
    // Red brand would otherwise share danger's meaning - "success" and "failed" must never share a color.
    expect(semantic.success).not.toBe(brand.base);
    expect(semantic.success).not.toBe(semantic.danger);
  });

  it('four seat states (free/selecting/held/sold) all differ', () => {
    // Spec redefined states by color (free=grey, selecting=blue, held=amber, sold=red): assert 4 distinct colors instead of pinning one value.
    const values = [seat.available, seat.selected, seat.held, seat.sold];
    expect(new Set(values).size).toBe(values.length);
  });

  it('text on brand passes contrast', () => {
    // Figma paired white with #1DE782 (~1.4:1). This test blocks going back.
    expect(contrast(brand.base, textOnBrand)).toBeGreaterThanOrEqual(4.5);
  });

  it.each(Object.entries(seatType))('seat label %s reads on its background', (_name, pair) => {
    expect(contrast(pair.bg, pair.fg)).toBeGreaterThanOrEqual(4.5);
  });

  it('brandAlpha derives from brand, never a hand-written copy', () => {
    // Blocks hand-written 'rgba(29, 231, 130, ...)' in theme/index.ts.
    const n = Number.parseInt(brand.base.slice(1), 16);
    const rgb = `${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}`;
    expect(brandAlpha(0.16)).toBe(`rgba(${rgb}, 0.16)`);
    expect(brandAlpha(1)).toBe(`rgba(${rgb}, 1)`);
  });

  it('bon loai ghe co bon mau nen khac nhau', () => {
    const backgrounds = Object.values(seatType).map((pair) => pair.bg);
    expect(new Set(backgrounds).size).toBe(backgrounds.length);
  });

  it.each(Object.entries(ratingColor))(
    'nhan do tuoi %s doc duoc tren nen cua no',
    (_name, pair) => {
      expect(contrast(pair.bg, pair.fg)).toBeGreaterThanOrEqual(4.5);
    }
  );

  it('nam muc do tuoi co nam mau nen khac nhau', () => {
    const backgrounds = Object.values(ratingColor).map((pair) => pair.bg);
    expect(new Set(backgrounds).size).toBe(backgrounds.length);
  });

  it('age-rating colors differ from brand', () => {
    // All 5 levels once shared brand.base - this test blocks going back.
    for (const pair of Object.values(ratingColor)) {
      expect(pair.bg.toLowerCase()).not.toBe(brand.base.toLowerCase());
    }
  });
});
