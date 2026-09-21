import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { cinemaGradientPanel } from '@/theme';
import { PATHS } from '@/routes/paths';

interface CustomerAuthShellProps {
  title: string;
  children: ReactNode;
  foot?: ReactNode;
}

/** Customer split-panel auth shell (Figma Sign In 77-626). Fixed dark-greeting/light-form halves regardless of customer light/dark; unlike AuthLayout's centered card. */
export const CustomerAuthShell = ({ title, children, foot }: CustomerAuthShellProps) => {
  const { t } = useTranslation();

  return (
    <div className="grid min-h-[calc(100svh_-_5rem)] flex-1 grid-cols-2 max-[820px]:grid-cols-1">
      {/* Viewport minus header (5rem) so both halves stretch evenly and the footer scrolls below. */}
      <aside
        className="flex flex-col justify-between px-10 py-9 text-white max-[820px]:hidden"
        style={{ backgroundColor: 'var(--cp-backdrop-base)', backgroundImage: cinemaGradientPanel }}
      >
        <Link
          to={PATHS.home}
          className="inline-flex items-center gap-2 text-xl font-bold text-white no-underline"
        >
          <span className="text-[22px] leading-none text-brand" aria-hidden="true">
            ◗
          </span>
          {t('customer.brand')}
        </Link>
        <p className="m-0 text-[40px] leading-[1.25] font-light italic">{t('customer.welcome')}</p>
      </aside>

      <main className="flex items-center justify-center bg-(--cp-surface-base) px-8 py-10">
        <div className="w-full max-w-[400px] text-[#141414]">
          <h1 className="mt-0 mb-5.5 text-[26px] font-bold">{title}</h1>
          {children}
          {foot ? <p className="mt-4 text-center text-[13px] text-[#666]">{foot}</p> : null}
        </div>
      </main>
    </div>
  );
};

export default CustomerAuthShell;
