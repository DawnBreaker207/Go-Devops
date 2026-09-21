import type { ReactNode } from 'react';
import { INK_75 } from '@/theme/customerTw';

/** Centered narrow frame for static pages. */
export const StaticPage = ({ children }: { children: ReactNode }) => (
  <div className="mx-auto max-w-[640px] px-0 py-6 pb-10 text-center">{children}</div>
);

export const StaticPageBody = ({ children }: { children: ReactNode }) => (
  <p className={`m-0 leading-relaxed ${INK_75}`}>{children}</p>
);

export default StaticPage;
