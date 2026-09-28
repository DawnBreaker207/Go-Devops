/// <reference types="node" />
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// No `@/index.css?raw`: vite.config sets test.css = false, so CSS imports are stubbed empty.
// Triple-slash reference scopes @types/node to this file instead of tsconfig.app.json (would leak Node globals into browser code).
const css = readFileSync(resolve(process.cwd(), 'src/index.css'), 'utf8');
import { distance, duration, easingCss, scale, stagger } from '@/motion';

// Forces src/motion.ts (TS) and :root in src/index.css to agree instead of drifting apart.
const cssVar = (name: string): string | undefined => {
  // Read only the first :root block; the @media reduced-motion block holds different values.
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
