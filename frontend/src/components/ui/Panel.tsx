import type { ReactNode } from 'react';
import { INK_BG_035, INK_BORDER_14 } from '@/theme/customerTw';

export const Panel = ({ children, className }: { children: ReactNode; className?: string }) => (
  <div
    className={[
      'mb-6 rounded-(--radius-card) border p-3.5 sm:p-4.5',
      INK_BORDER_14,
      INK_BG_035,
      className ?? '',
    ]
      .filter(Boolean)
      .join(' ')}
  >
    {children}
  </div>
);

export default Panel;
