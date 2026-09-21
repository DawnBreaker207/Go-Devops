import { FOCUS_RING, HOVER_INK_BORDER, INK, INK_BORDER_35 } from '@/theme/customerTw';

export type ButtonVariant = 'primary' | 'ghost' | 'danger';

const BASE =
  'inline-flex items-center justify-center gap-2 min-h-10 px-5 rounded-lg border border-transparent ' +
  'text-sm font-semibold no-underline cursor-pointer transition-[transform,background-color,border-color,color] ' +
  'duration-fast ease-out active:scale-[var(--motion-scale-press)] disabled:cursor-not-allowed disabled:opacity-45 ' +
  `${FOCUS_RING}`;

const VARIANT: Record<ButtonVariant, string> = {
  // No Tailwind variant for `:hover:not(:disabled)` - approximate: disabled already dims + unclickable, so hover color matters less there.
  primary: 'bg-brand text-on-brand hover-fine:bg-brand-hover',
  ghost: `bg-transparent ${INK_BORDER_35} ${INK} ${HOVER_INK_BORDER}`,
  danger: 'bg-danger text-white',
};

/** Customer button classes shared by button and link variants. */
export const buttonClassName = (
  variant: ButtonVariant,
  opts?: { block?: boolean; className?: string }
): string =>
  [BASE, VARIANT[variant], opts?.block ? 'w-full' : '', opts?.className ?? '']
    .filter(Boolean)
    .join(' ');
