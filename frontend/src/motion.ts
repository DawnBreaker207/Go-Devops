export const duration = {
  instant: 100,
  fast: 150,
  base: 200,
  moderate: 300,
  slow: 450,
  decorative: 600,
} as const;

export type DurationToken = keyof typeof duration;

export const seconds = (token: DurationToken) => duration[token] / 1000;

export const easing = {
  out: [0.22, 1, 0.36, 1],
  in: [0.32, 0, 0.67, 0],
  inOut: [0.65, 0, 0.35, 1],
  linear: [0, 0, 1, 1],
} as const;

export type EasingToken = keyof typeof easing;

export const easingCss: Record<EasingToken, string> = {
  out: 'cubic-bezier(0.22, 1, 0.36, 1)',
  in: 'cubic-bezier(0.32, 0, 0.67, 0)',
  inOut: 'cubic-bezier(0.65, 0, 0.35, 1)',
  linear: 'linear',
};

export const spring = {
  base: { type: 'spring', duration: seconds('base'), bounce: 0.15 },
  flat: { type: 'spring', duration: seconds('moderate'), bounce: 0 },
} as const;

export const distance = {
  lift: 4,
  shift: 12,
  reveal: 16,
} as const;

export const scale = {
  enter: 0.96,
  press: 0.98,
  zoom: 1.05,
  pop: 1.08,
} as const;

export const stagger = {
  step: 50,
  max: 6,
} as const;

export const staggerDelay = (index: number) => Math.min(index, stagger.max - 1) * stagger.step;

export const cssTransition = (
  properties: string[],
  token: DurationToken = 'base',
  ease: EasingToken = 'out'
) => properties.map((p) => `${p} ${duration[token]}ms ${easingCss[ease]}`).join(', ');

export const motionTokens = { duration, easing, easingCss, spring, distance, scale, stagger };
