import type { ReactNode } from 'react';
import { INK_60 } from '@/theme/customerTw';

export interface SectionHeadProps {
  eyebrow?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
}

/** Centered content header: optional eyebrow + title + short subtitle. */
export const SectionHead = ({ eyebrow, title, subtitle }: SectionHeadProps) => (
  <div className="px-0 py-5 pb-8 text-center">
    {eyebrow ? (
      <span className="mb-2.5 inline-block rounded-full border border-[rgb(var(--cp-ink-rgb))]/18 bg-[rgb(var(--cp-ink-rgb))]/5 px-3.5 py-1 text-xs font-bold tracking-wider text-brand uppercase">
        {eyebrow}
      </span>
    ) : null}
    <h1 className="m-0 mb-2 text-2xl font-extrabold tracking-tight sm:text-[32px]">{title}</h1>
    {subtitle ? <p className={`mx-auto max-w-[460px] text-sm ${INK_60}`}>{subtitle}</p> : null}
  </div>
);

export default SectionHead;
