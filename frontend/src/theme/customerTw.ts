/**
/** Shared Tailwind strings for the customer zone. Must stay STATIC: Tailwind scans text by regex, dynamic builders are never detected. "ink" follows light/dark via --cp-ink-rgb; no hex here. */
export const INK = 'text-[rgb(var(--cp-ink-rgb))]';
export const INK_85 = 'text-[rgb(var(--cp-ink-rgb))]/85';
export const INK_80 = 'text-[rgb(var(--cp-ink-rgb))]/80';
export const INK_75 = 'text-[rgb(var(--cp-ink-rgb))]/75';
export const INK_65 = 'text-[rgb(var(--cp-ink-rgb))]/65';
export const INK_62 = 'text-[rgb(var(--cp-ink-rgb))]/62';
export const INK_60 = 'text-[rgb(var(--cp-ink-rgb))]/60';
export const INK_55 = 'text-[rgb(var(--cp-ink-rgb))]/55';
export const INK_40 = 'text-[rgb(var(--cp-ink-rgb))]/40';
export const INK_35 = 'text-[rgb(var(--cp-ink-rgb))]/35';

export const INK_BG = 'bg-[rgb(var(--cp-ink-rgb))]';
export const INK_BG_03 = 'bg-[rgb(var(--cp-ink-rgb))]/[0.03]';
export const INK_BG_035 = 'bg-[rgb(var(--cp-ink-rgb))]/[0.035]';
export const INK_BG_04 = 'bg-[rgb(var(--cp-ink-rgb))]/[0.04]';
export const INK_BG_05 = 'bg-[rgb(var(--cp-ink-rgb))]/5';
export const INK_BG_06 = 'bg-[rgb(var(--cp-ink-rgb))]/6';
export const INK_BG_07 = 'bg-[rgb(var(--cp-ink-rgb))]/[0.07]';
export const INK_BG_08 = 'bg-[rgb(var(--cp-ink-rgb))]/8';

export const INK_BORDER_10 = 'border-[rgb(var(--cp-ink-rgb))]/10';
export const INK_BORDER_14 = 'border-[rgb(var(--cp-ink-rgb))]/14';
export const INK_BORDER_16 = 'border-[rgb(var(--cp-ink-rgb))]/16';
export const INK_BORDER_18 = 'border-[rgb(var(--cp-ink-rgb))]/18';
export const INK_BORDER_20 = 'border-[rgb(var(--cp-ink-rgb))]/20';
export const INK_BORDER_22 = 'border-[rgb(var(--cp-ink-rgb))]/22';
export const INK_BORDER_25 = 'border-[rgb(var(--cp-ink-rgb))]/25';
export const INK_BORDER_28 = 'border-[rgb(var(--cp-ink-rgb))]/28';
export const INK_BORDER_35 = 'border-[rgb(var(--cp-ink-rgb))]/35';

// `hover-fine:` fused with another utility is invisible to Tailwind (the fused
// string only exists at runtime), so pre-fuse full strings below.
export const HOVER_INK = 'hover-fine:text-[rgb(var(--cp-ink-rgb))]';
export const HOVER_INK_BORDER = 'hover-fine:border-[rgb(var(--cp-ink-rgb))]';
export const HOVER_INK_BG_06 = 'hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6';

/** Single-property transition on bridged --motion-* tokens; never `transition-all`. */
export const TRANSITION_FAST = 'transition-colors duration-fast ease-out';
export const TRANSITION_BORDER_FAST = 'transition-[border-color] duration-fast ease-out';
export const TRANSITION_TRANSFORM_FAST = 'transition-transform duration-fast ease-out';

/** Shared focus ring in brand color. */
export const FOCUS_RING =
  'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand';

/** Shared PosterCard grid for home and /films. */
export const BROWSE_GRID = 'grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4';
