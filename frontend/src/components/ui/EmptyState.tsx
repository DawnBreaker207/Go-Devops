import type { ReactNode } from 'react';
import { INK_60 } from '@/theme/customerTw';

/** Centered empty placeholder. */
export const EmptyState = ({ children }: { children: ReactNode }) => (
  <p className={`px-4 py-12 text-center ${INK_60}`}>{children}</p>
);

export default EmptyState;
