/// <reference types="node" />
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// Doc thang file CSS tu dia. Khong dung `@/index.css?raw` vi vite.config.ts dat
// test.css = false, nen moi import CSS (ke ca ?raw) deu bi stub thanh rong.
// Triple-slash reference o tren de @types/node co hieu luc rieng cho file nay,
// khong phai them "node" vao types cua tsconfig.app.json (se ro ri Node global
// vao code chay tren browser). vitest chay voi cwd = goc repo.
const css = readFileSync(resolve(process.cwd(), 'src/index.css'), 'utf8');
import { distance, duration, easingCss, scale, stagger } from '@/motion';

// Motion token duoc khai bao hai noi: src/motion.ts (cho TS) va :root trong
// src/index.css (cho CSS). Test nay la thu ep hai ben khong lech nhau - neu ai do
// sua mot ben, test do se fail thay vi de UI chay hai toc do khac nhau.
const cssVar = (name: string): string | undefined => {
  // Chi doc khoi :root dau tien; khoi trong @media reduced-motion co gia tri khac.
  const root = css.slice(css.indexOf(':root'), css.indexOf('@media'));
  return root.match(new RegExp(`--${name}:\\s*([^;]+);`))?.[1].trim();
};

describe('motion token: motion.ts va index.css phai khop', () => {
  it.each(Object.entries(duration))('duration.%s', (name, ms) => {
    expect(cssVar(`motion-duration-${name}`)).toBe(`${ms}ms`);
  });

  it.each([
    ['out', 'motion-ease-out'],
    ['in', 'motion-ease-in'],
    ['inOut', 'motion-ease-in-out'],
    ['linear', 'motion-ease-linear'],
  ] as const)('easing.%s', (token, varName) => {
    expect(cssVar(varName)).toBe(easingCss[token]);
  });

  it.each(Object.entries(distance))('distance.%s', (name, px) => {
    expect(cssVar(`motion-distance-${name}`)).toBe(`${px}px`);
  });

  it.each(Object.entries(scale))('scale.%s', (name, value) => {
    expect(cssVar(`motion-scale-${name}`)).toBe(String(value));
  });

  it('stagger', () => {
    expect(cssVar('motion-stagger-step')).toBe(`${stagger.step}ms`);
    expect(cssVar('motion-stagger-max')).toBe(String(stagger.max));
  });
});

describe('motion token: rang buoc cua he thong', () => {
  it('duration tang dan theo dung thu tu', () => {
    const order = [
      duration.instant,
      duration.fast,
      duration.base,
      duration.moderate,
      duration.slow,
      duration.decorative,
    ];
    expect(order).toEqual([...order].sort((a, b) => a - b));
  });

  it('reduced motion bo dich chuyen va zoom', () => {
    const reduced = css.slice(css.indexOf('@media (prefers-reduced-motion: reduce)'));
    expect(reduced).toMatch(/--motion-distance-lift:\s*0px/);
    expect(reduced).toMatch(/--motion-distance-shift:\s*0px/);
    expect(reduced).toMatch(/--motion-distance-reveal:\s*0px/);
    expect(reduced).toMatch(/--motion-scale-zoom:\s*1;/);
    expect(reduced).toMatch(/--motion-scale-pop:\s*1;/);
  });

  it('khong duration nao vuot fast khi reduced motion', () => {
    const reduced = css.slice(css.indexOf('@media (prefers-reduced-motion: reduce)'));
    const values = [...reduced.matchAll(/--motion-duration-[a-z]+:\s*(\d+)ms/g)].map((m) =>
      Number(m[1])
    );
    expect(values.length).toBeGreaterThan(0);
    expect(Math.max(...values)).toBeLessThanOrEqual(duration.fast);
  });
});

describe('stagger', () => {
  it('toi da 6 phan tu, phan con lai hien cung luc', async () => {
    const { staggerDelay } = await import('@/motion');
    expect(staggerDelay(0)).toBe(0);
    expect(staggerDelay(5)).toBe(250);
    expect(staggerDelay(6)).toBe(250);
    expect(staggerDelay(99)).toBe(250);
  });
});
