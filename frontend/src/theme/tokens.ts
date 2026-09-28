export const brand = {
  base: '#D33B56',
  hover: '#DC6278',
  active: '#B33249',
  soft: '#F8E2E6',
  softer: '#FCF1F3',
} as const;

export const textOnBrand = '#FFFFFF';

export const brandAlpha = (alpha: number): string => {
  const n = Number.parseInt(brand.base.slice(1), 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
};

export const cinemaBackdrop = {
  base: '#060203',
  mid: '#20090D',
  glow: '#4A151E',
} as const;

export const customerLightCanvas = '#F7F6F4';

export const semantic = {
  danger: '#D22F27',
  warning: '#D97706',
  success: '#1DE782',
  info: '#2563EB',
} as const;

export const surface = {
  light: {
    base: '#FFFFFF',
    sunken: '#FAFAFA',
    border: '#BFBFBF',
    borderSubtle: '#EBEBEB',
    iconMuted: '#767676',
  },
  dark: {
    base: '#141414',
    sunken: '#0F0F0F',
    raised: '#1F1F1F',
    border: '#303030',
    borderSubtle: '#262626',
  },
} as const;

const neutralLabelGrey = '#3A3F3C';

export const seat = {
  available: '#D1D5DB',
  availableText: '#1F2937',
  availableHover: '#22C55E',
  selected: '#2563EB',
  selectedText: '#FFFFFF',
  sold: '#DC2626',
  soldText: '#FFFFFF',
  held: '#D97706',
  heldText: '#1A1200',
  gap: 'transparent',
  screen: '#FFFFFF',
} as const;

export const seatType = {
  standard: { bg: '#EDF0EE', fg: neutralLabelGrey },
  vip: { bg: brand.soft, fg: '#720016' },
  couple: { bg: '#FDE8CE', fg: '#7A4405' },
  recliner: { bg: '#DDE7FE', fg: '#1E3A8A' },
} as const;

export const fontFamily = {
  display: "'Be Vietnam Pro', 'Segoe UI', system-ui, -apple-system, sans-serif",
  body: "'IBM Plex Sans', 'Segoe UI', system-ui, -apple-system, sans-serif",
} as const;

export const ratingColor = {
  p: { bg: '#2E7D32', fg: '#FFFFFF' },
  k: { bg: '#1D4ED8', fg: '#FFFFFF' },
  t13: { bg: '#FDB813', fg: '#402D00' },
  t16: { bg: '#C2410C', fg: '#FFFFFF' },
  t18: { bg: '#8B0000', fg: '#FFFFFF' },
} as const;

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

export const cinemaGradient =
  `radial-gradient(120% 90% at 8% 40%, ${cinemaBackdrop.glow} 0%, ` +
  `${cinemaBackdrop.mid} 35%, ${cinemaBackdrop.base} 75%)`;

export const cinemaGradientPanel =
  `linear-gradient(160deg, ${cinemaBackdrop.base} 0%, ` +
  `${cinemaBackdrop.mid} 70%, ${cinemaBackdrop.glow} 100%)`;
