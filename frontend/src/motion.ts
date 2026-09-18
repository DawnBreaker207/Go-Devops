/**
 * Motion tokens - nguon duy nhat cho moi hieu ung trong app.
 *
 * Cac gia tri nay duoc mirror thanh CSS custom property trong src/index.css (khoi
 * `:root` co comment MOTION TOKENS). Test src/__tests__/motion-tokens.test.ts doc ca
 * hai file va fail neu chung lech nhau, nen sua o mot noi thi phai sua ca noi kia.
 *
 * Component KHONG duoc hardcode so: dung `motion.duration.base` trong TS, hoac
 * `var(--motion-duration-base)` trong CSS.
 */

/** Thoi luong, mili-giay. `decorative` chi danh cho hero/nen, khong dung cho UI feedback. */
export const duration = {
  instant: 100,
  fast: 150,
  base: 200,
  moderate: 300,
  slow: 450,
  decorative: 600,
} as const;

export type DurationToken = keyof typeof duration;

/** Motion (thu vien JS) nhan duration bang giay. */
export const seconds = (token: DurationToken) => duration[token] / 1000;

/**
 * Easing. Dang mang dung cho Motion (`transition={{ ease: easing.out }}`),
 * dang chuoi dung cho CSS (`transition-timing-function`).
 */
export const easing = {
  /** Vao man hinh, hover. */
  out: [0.22, 1, 0.36, 1],
  /** Roi man hinh. */
  in: [0.32, 0, 0.67, 0],
  /** Di chuyen trong man hinh. */
  inOut: [0.65, 0, 0.35, 1],
  /** Chi cho spinner, progress, vong lap. */
  linear: [0, 0, 1, 1],
} as const;

export type EasingToken = keyof typeof easing;

export const easingCss: Record<EasingToken, string> = {
  out: 'cubic-bezier(0.22, 1, 0.36, 1)',
  in: 'cubic-bezier(0.32, 0, 0.67, 0)',
  inOut: 'cubic-bezier(0.65, 0, 0.35, 1)',
  linear: 'linear',
};

/**
 * Spring cho keo tha va doi layout. Khong nay hoac nay rat nhe: bounce <= 0.15.
 * Motion: `transition={{ type: 'spring', ...spring.base }}`.
 */
export const spring = {
  /** Mac dinh cho layout animation va chon/bo chon. */
  base: { type: 'spring', duration: seconds('base'), bounce: 0.15 },
  /** Khong nay - dung khi phan tu lon di chuyen. */
  flat: { type: 'spring', duration: seconds('moderate'), bounce: 0 },
} as const;

/** Khoang dich chuyen, pixel. */
export const distance = {
  /** Nang card/nut khi hover. */
  lift: 4,
  /** Truot ngang/doc cho toast, buoc checkout. */
  shift: 12,
  /** Scroll reveal. */
  reveal: 16,
} as const;

/** He so scale. */
export const scale = {
  /** Trang thai bat dau khi mot phan tu xuat hien. */
  enter: 0.96,
  /** Trang thai dang nhan nut. */
  press: 0.98,
  /** Zoom anh trong khung overflow hidden. */
  zoom: 1.05,
  /** Nhan manh ngan, vd ghe vua duoc chon. */
  pop: 1.08,
} as const;

/** Stagger: 50ms moi phan tu, toi da 6 phan tu roi phan con lai hien cung luc. */
export const stagger = {
  step: 50,
  max: 6,
} as const;

/** Do tre cua phan tu thu `index` trong mot danh sach stagger, mili-giay. */
export const staggerDelay = (index: number) => Math.min(index, stagger.max - 1) * stagger.step;

/** `transition` cho CSS: shorthand dung token, khong bao gio dung `all`. */
export const cssTransition = (
  properties: string[],
  token: DurationToken = 'base',
  ease: EasingToken = 'out'
) => properties.map((p) => `${p} ${duration[token]}ms ${easingCss[ease]}`).join(', ');

export const motionTokens = { duration, easing, easingCss, spring, distance, scale, stagger };
