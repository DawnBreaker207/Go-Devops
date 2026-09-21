/** Animation single source, mirrored to --motion-* CSS (parity test fails on drift). Never hardcode: tokens in TS, var(--motion-*) in CSS. */

/** Durations, ms. `decorative` is hero/background only, not UI feedback. */
export const duration = {
  instant: 100,
  fast: 150,
  base: 200,
  moderate: 300,
  slow: 450,
  decorative: 600,
} as const;

export type DurationToken = keyof typeof duration;

/** Motion lib takes seconds. */
export const seconds = (token: DurationToken) => duration[token] / 1000;

/** Easing: array for Motion, string for CSS. */
export const easing = {
  /** Enter, hover. */
  out: [0.22, 1, 0.36, 1],
  /** Exit. */
  in: [0.32, 0, 0.67, 0],
  /** Move within screen. */
  inOut: [0.65, 0, 0.35, 1],
  /** Spinners, progress, loops only. */
  linear: [0, 0, 1, 1],
} as const;

export type EasingToken = keyof typeof easing;

export const easingCss: Record<EasingToken, string> = {
  out: 'cubic-bezier(0.22, 1, 0.36, 1)',
  in: 'cubic-bezier(0.32, 0, 0.67, 0)',
  inOut: 'cubic-bezier(0.65, 0, 0.35, 1)',
  linear: 'linear',
};

/** Springs for drag and layout. Keep bounce <= 0.15. */
export const spring = {
  /** Default for layout and select/deselect. */
  base: { type: 'spring', duration: seconds('base'), bounce: 0.15 },
  /** Flat, for large moving elements. */
  flat: { type: 'spring', duration: seconds('moderate'), bounce: 0 },
} as const;

/** Offsets, px. */
export const distance = {
  /** Lift card/button on hover. */
  lift: 4,
  /** Slide for toast, checkout steps. */
  shift: 12,
  /** Scroll reveal. */
  reveal: 16,
} as const;

/** Scale factors. */
export const scale = {
  /** Initial state when an element appears. */
  enter: 0.96,
  /** Pressed button. */
  press: 0.98,
  /** Image zoom inside overflow-hidden frame. */
  zoom: 1.05,
  /** Brief pop, e.g. just-selected seat. */
  pop: 1.08,
} as const;

/** Stagger: 50ms per item, first 6 only, rest appear together. */
export const stagger = {
  step: 50,
  max: 6,
} as const;

/** Stagger delay for `index`, ms. */
export const staggerDelay = (index: number) => Math.min(index, stagger.max - 1) * stagger.step;

/** CSS transition shorthand from tokens; never `all`. */
export const cssTransition = (
  properties: string[],
  token: DurationToken = 'base',
  ease: EasingToken = 'out'
) => properties.map((p) => `${p} ${duration[token]}ms ${easingCss[ease]}`).join(', ');

export const motionTokens = { duration, easing, easingCss, spring, distance, scale, stagger };
