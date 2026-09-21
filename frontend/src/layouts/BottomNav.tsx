import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PATHS, accountTicketsPath } from '@/routes/paths';
import { useHasRole } from '@/hooks/useHasRole';
import { INK_60 } from '@/theme/customerTw';

/** 5 mobile items (<900px). "My tickets" is customer-only; the other 4 always show so the bar never jumps on login/logout. */
export const BottomNav = () => {
  const { t } = useTranslation();
  const isCustomer = useHasRole('customer');

  const itemClass = ({ isActive }: { isActive: boolean }) =>
    [
      'flex min-w-0 flex-1 flex-col items-center justify-center gap-0.5 px-0.5 py-1.5 text-center text-[11px] leading-[1.2] font-semibold no-underline',
      'transition-colors duration-fast ease-out',
      'focus-visible:rounded-lg focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-brand',
      isActive ? 'text-brand' : INK_60,
    ].join(' ');

  return (
    <nav
      className="hidden max-[900px]:fixed max-[900px]:inset-x-0 max-[900px]:bottom-0 max-[900px]:z-30 max-[900px]:flex max-[900px]:items-stretch max-[900px]:justify-around max-[900px]:h-15 max-[900px]:border-t max-[900px]:border-(--cp-chrome-border) max-[900px]:bg-(--cp-chrome-bg-strong) max-[900px]:pb-[env(safe-area-inset-bottom,0px)] max-[900px]:backdrop-blur-md"
      aria-label={t('customer.bottomNavLabel')}
    >
      <NavLink to={PATHS.home} end className={itemClass}>
        <span className="text-lg leading-none" aria-hidden="true">
          🏠
        </span>
        {t('customer.navHome')}
      </NavLink>
      <NavLink to={PATHS.cinemaInfo} className={itemClass}>
        <span className="text-lg leading-none" aria-hidden="true">
          🎬
        </span>
        {t('customer.bottomNavCinema')}
      </NavLink>
      <NavLink to={isCustomer ? accountTicketsPath() : PATHS.customerLogin} className={itemClass}>
        <span className="text-lg leading-none" aria-hidden="true">
          🎟️
        </span>
        {t('customer.myTickets')}
      </NavLink>
      <NavLink to={PATHS.offers} className={itemClass}>
        <span className="text-lg leading-none" aria-hidden="true">
          🏷️
        </span>
        {t('customer.bottomNavOffers')}
      </NavLink>
      <NavLink to={isCustomer ? PATHS.account : PATHS.customerLogin} className={itemClass}>
        <span className="text-lg leading-none" aria-hidden="true">
          👤
        </span>
        {t('customer.bottomNavAccount')}
      </NavLink>
    </nav>
  );
};

export default BottomNav;
