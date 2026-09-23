import { INK, INK_65, INK_BG_03, INK_BORDER_22 } from '@/theme/customerTw';

/** Shared classes for one showtime button. */
export const showtimeCardClass = [
  'relative flex w-full cursor-pointer flex-col gap-0.5 rounded-[10px] border px-3.5 py-3 text-left font-[inherit] no-underline',
  INK_BORDER_22,
  INK_BG_03,
  INK,
  'transition-[border-color,transform] duration-fast ease-out',
  "after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:shadow-[0_10px_24px_rgba(0,0,0,0.35)] after:transition-opacity after:duration-fast after:ease-out after:content-['']",
  'hover-fine:-translate-y-(--motion-distance-lift) hover-fine:border-brand hover-fine:after:opacity-100',
  'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand focus-visible:after:opacity-100',
].join(' ');

export const showtimeTimeClass = 'text-lg font-bold';
export const showtimeHallClass = `text-xs ${INK_65}`;
export const showtimePriceClass = 'text-xs font-semibold text-brand';
