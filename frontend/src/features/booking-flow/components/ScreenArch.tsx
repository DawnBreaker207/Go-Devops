import { useTranslation } from 'react-i18next';
import { INK_60 } from '@/theme/customerTw';

/** Arched screen (depth gradient + glass layer), theme-reactive. Replaces the old flat bar. */
export const ScreenArch = () => {
  const { t } = useTranslation();
  return (
    <div className="mx-auto flex w-[min(560px,92%)] flex-col items-center">
      <svg viewBox="0 0 560 60" className="w-full text-[rgb(var(--cp-ink-rgb))]" aria-hidden="true">
        <defs>
          <linearGradient id="cp-screen-arch-fade" x1="0" x2="1" y1="0" y2="0">
            <stop offset="0%" stopColor="currentColor" stopOpacity="0" />
            <stop offset="15%" stopColor="currentColor" stopOpacity="0.9" />
            <stop offset="85%" stopColor="currentColor" stopOpacity="0.9" />
            <stop offset="100%" stopColor="currentColor" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path
          d="M10 50 Q280 -10 550 50"
          fill="none"
          stroke="url(#cp-screen-arch-fade)"
          strokeWidth="4"
          strokeLinecap="round"
        />
      </svg>
      <div className="-mt-1 h-4 w-[85%] rounded-[100%] bg-[rgb(var(--cp-ink-rgb))]/10 blur-[6px]" />
      <div className={`mt-2 text-[11px] tracking-[3px] uppercase ${INK_60}`}>
        {t('customer.screen')}
      </div>
    </div>
  );
};

export default ScreenArch;
