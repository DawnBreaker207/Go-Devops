import type { ReactNode } from 'react';

export type NoticeVariant = 'error' | 'info' | 'success';

export interface NoticeProps {
  variant: NoticeVariant;
  children: ReactNode;
  className?: string;
  role?: 'alert' | 'status';
  /** Stacked (column, left-aligned) instead of the default row; a separate branch so flex-row/flex-col utilities never collide on one element. */
  stack?: boolean;
}

const VARIANT: Record<NoticeVariant, string> = {
  error: 'border-danger bg-danger/10 text-[#8c1f19]',
  info: 'border-[rgb(var(--cp-ink-rgb))]/28 bg-[rgb(var(--cp-ink-rgb))]/6 text-[rgb(var(--cp-ink-rgb))]',
  // brand-softer is a FIXED light bg, so text must be fixed-dark too; white on it was unreadable (same bug class fixed in theme/index.ts Menu item).
  success: 'border-brand-active bg-brand-softer text-[#1a1a1a]',
};

/** Shared customer notice (error/success/info). */
export const Notice = ({ variant, children, className, role = 'alert', stack }: NoticeProps) => (
  <div
    role={role}
    className={[
      stack
        ? 'flex flex-col items-start gap-2 rounded-lg border px-3.5 py-3 text-sm leading-relaxed'
        : 'flex gap-2.5 rounded-lg border px-3.5 py-3 text-sm leading-relaxed',
      VARIANT[variant],
      className ?? '',
    ]
      .filter(Boolean)
      .join(' ')}
  >
    {children}
  </div>
);

export default Notice;
