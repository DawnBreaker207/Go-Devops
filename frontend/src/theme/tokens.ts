/** Design tokens. MEASURED = frame pixels; DERIVED = inferred (formula noted). Single source of truth mirrored by theme CSS (parity test fails on drift). No antd/React imports. */

/* -------------------------------------------------------------------------- */
/* 1. Brand                                                                    */
/* -------------------------------------------------------------------------- */

/** Brand red (deliberate detour from the source Figma) - one color for admin and customer. Variants below mix per the old formula (20% white / 15% black / 85% / 93% white). */
export const brand = {
  /** Brand red. */
  base: '#E4002B',
  /** DERIVED - #E4002B + 20% white, for hover. */
  hover: '#E93355',
  /** DERIVED - #E4002B + 15% black, for :active and badge borders. */
  active: '#C20025',
  /** DERIVED - #E4002B + 85% white, selected-menu background in sider. */
  soft: '#FBD9DF',
  /** DERIVED - #E4002B + 93% white, paler than `soft` for tags/light selections. */
  softer: '#FDEDF0',
} as const;

/** White text on brand.base (~4.85:1, passes WCAG 4.5:1 - see design-tokens.test.ts). */
export const textOnBrand = '#FFFFFF';

/** brand.base at a given alpha - computed from source so it can't drift (parity test can't see hand-written rgba strings). */
export const brandAlpha = (alpha: number): string => {
  const n = Number.parseInt(brand.base.slice(1), 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
};

/* -------------------------------------------------------------------------- */
/* 2. Customer dark backdrop                                                     */
/* -------------------------------------------------------------------------- */

/** Dark customer backdrop: brand.base toward black (DERIVED, base -> glow). */
export const cinemaBackdrop = {
  /** DERIVED - brand.base + 97% black. Darkest point, near-black. */
  base: '#070001',
  /** DERIVED - brand.base + 85% black. Mid glow. */
  mid: '#220006',
  /** DERIVED - brand.base + 65% black. Brightest glow, and toolbar bg under Select Seat. */
  glow: '#50000F',
} as const;

/** Customer light canvas (detour: #f7f6f4). Browse screens + header/nav/footer only; seats/payment/tickets are ALWAYS dark since white seats only read on dark. */
export const customerLightCanvas = '#F7F6F4';

/* -------------------------------------------------------------------------- */
/* 3. Semantic colors                                                            */
/* -------------------------------------------------------------------------- */

export const semantic = {
  /** MEASURED - Admin Movie delete button (merged with customer Logout red). */
  danger: '#D22F27',
  /** DERIVED - no warning state in design; amber readable on light and dark. */
  warning: '#D97706',
  /** Old brand green - split from brand.base so "success" orders never share color with "failed" ones. */
  success: '#1DE782',
  /** DERIVED - neutral blue for non-bad news. */
  info: '#2563EB',
} as const;

/* -------------------------------------------------------------------------- */
/* 4. Surfaces and borders                                                       */
/* -------------------------------------------------------------------------- */

export const surface = {
  light: {
    /** MEASURED - every admin frame. */
    base: '#FFFFFF',
    /** DERIVED - off-white so content cards still read; Figma uses full white + stronger border. */
    sunken: '#FAFAFA',
    /** MEASURED - Admin Movie table borders. */
    border: '#BFBFBF',
    /** DERIVED - lighter border for cards/inputs (#BFBFBF too heavy there). */
    borderSubtle: '#EBEBEB',
    /** MEASURED - ghost-button icons/borders in Admin Movie. */
    iconMuted: '#767676',
  },
  dark: {
    /** DERIVED - antd darkAlgorithm base; admin dark never uses customer blue-black. */
    base: '#141414',
    sunken: '#0F0F0F',
    raised: '#1F1F1F',
    border: '#303030',
    borderSubtle: '#262626',
  },
} as const;

/* -------------------------------------------------------------------------- */
/* 5. Seats                                                                      */
/* -------------------------------------------------------------------------- */

/** Seat states beyond free/selecting: sold = dark grey (unclickable); held = amber (may release); gap = transparent cell. */
/** Grey seat-type label for "standard" - SEPARATE from `seat.sold` (type vs status axes; sharing broke when sold turned red). */
const neutralLabelGrey = '#3A3F3C';

export const seat = {
  /** Pale grey per new seat-picker spec. */
  available: '#D1D5DB',
  availableText: '#1F2937',
  /** Free-seat hover - green, distinct from selected. */
  availableHover: '#22C55E',
  /** Blue (spec: free=grey, SELECTING=blue, held=orange, sold=red). */
  selected: '#2563EB',
  selectedText: '#FFFFFF',
  /** Red (new spec). */
  sold: '#DC2626',
  soldText: '#FFFFFF',
  /** DERIVED */
  held: '#D97706',
  heldText: '#1A1200',
  /** DERIVED - grid holes, not seats. */
  gap: 'transparent',
  /** MEASURED - the "X" bar for the cinema screen in Select Seat. */
  screen: '#FFFFFF',
} as const;

/* -------------------------------------------------------------------------- */
/* 6. Seat kinds - admin grid editor                                             */
/* -------------------------------------------------------------------------- */

/** Exactly 4 seat kinds, no listing endpoint, so FE keeps its own. All DERIVED. LIGHT admin surfaces about KIND (vs `seat`: dark customer STATUS) - never mix. */
export const seatType = {
  /** DERIVED - neutral grey, the default kind. Text uses `neutralLabelGrey` (>= 4.5:1; #767676 is 3.96:1, `seat.sold` is red now). */
  standard: { bg: '#EDF0EE', fg: neutralLabelGrey },
  /** bg = brand.soft; dark-red fg (~9.3:1). */
  vip: { bg: brand.soft, fg: '#720016' },
  /** DERIVED - semantic.warning tint, readable with dark-brown text. */
  couple: { bg: '#FDE8CE', fg: '#7A4405' },
  /** DERIVED - semantic.info tint. */
  recliner: { bg: '#DDE7FE', fg: '#1E3A8A' },
} as const;

/* -------------------------------------------------------------------------- */
/* 7. Typefaces                                                                  */
/* -------------------------------------------------------------------------- */

/** "Two-Family Rule": `display` for big titles, `body` for content - customer zone only (admin keeps Inter). Unrelated to `--font-mono` (figures only). */
export const fontFamily = {
  display: "'Be Vietnam Pro', 'Segoe UI', system-ui, -apple-system, sans-serif",
  body: "'IBM Plex Sans', 'Segoe UI', system-ui, -apple-system, sans-serif",
} as const;

/* -------------------------------------------------------------------------- */
/* 8. Movie age-rating badges (age_rating)                                       */
/* -------------------------------------------------------------------------- */

/** Per-level age-rating colors (enum untouched): rising-restriction ramp P..T18, never brand red. Every pair >= 4.5:1 light and dark. */
export const ratingColor = {
  /** P - all ages. */
  p: { bg: '#2E7D32', fg: '#FFFFFF' },
  /** K - under 13 with an adult. */
  k: { bg: '#1D4ED8', fg: '#FFFFFF' },
  /** T13 - 13+. */
  t13: { bg: '#FDB813', fg: '#402D00' },
  /** T16 - 16+. */
  t16: { bg: '#C2410C', fg: '#FFFFFF' },
  /** T18 - 18+, most restricted. */
  t18: { bg: '#8B0000', fg: '#FFFFFF' },
} as const;

/* -------------------------------------------------------------------------- */
/* 9. Bundle                                                                     */
/* -------------------------------------------------------------------------- */

export const tokens = {
  brand,
  textOnBrand,
  cinemaBackdrop,
  customerLightCanvas,
  semantic,
  surface,
  seat,
  seatType,
  fontFamily,
  ratingColor,
} as const;

/** Customer-zone gradient for <CustomerLayout>; never admin (Figma gives admin flat white). */
export const cinemaGradient =
  `radial-gradient(120% 90% at 8% 40%, ${cinemaBackdrop.glow} 0%, ` +
  `${cinemaBackdrop.mid} 35%, ${cinemaBackdrop.base} 75%)`;

/** Portrait variant for the sign-in/up split screen: same ramp as backdrop, bright at the bottom; only direction is DERIVED. */
export const cinemaGradientPanel =
  `linear-gradient(160deg, ${cinemaBackdrop.base} 0%, ` +
  `${cinemaBackdrop.mid} 70%, ${cinemaBackdrop.glow} 100%)`;
