import { Suspense, useState } from 'react';
import { Link, NavLink, Outlet, useMatches } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Loading from '@/components/Loading';
import BottomNav from './BottomNav';
import AvatarSwitcher from './AvatarSwitcher';
import Footer from './Footer';
import { useAppStore } from '@/stores/appStore';
import { cinemaGradient } from '@/theme';
import { PATHS } from '@/routes/paths';
import { INK, INK_80, INK_BORDER_10 } from '@/theme/customerTw';

/** 4 nav items shared by desktop/mobile ("Showtimes by cinema" dropped: no page yet). */
const NAV_ITEMS: Array<{ to: string; labelKey: string }> = [
  { to: PATHS.films, labelKey: 'customer.navMovies' },
  { to: PATHS.cinemaInfo, labelKey: 'customer.navCinema' },
  { to: PATHS.pricing, labelKey: 'customer.navPricing' },
  { to: PATHS.offers, labelKey: 'customer.navOffersNews' },
];

const MenuIcon = ({ open }: { open: boolean }) => (
  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    {open ? (
      <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    ) : (
      <path
        d="M4 7h16M4 12h16M4 17h16"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    )}
  </svg>
);

/** Per-route layout flags consumed below. */
interface RouteHandle {
  /** Account screen (split panel) needs full bleed, free from <main> max-width/padding. */
  fullBleed?: boolean;
  /** White seats (`seat.available`) only stand out on dark, so this screen stays dark regardless. */
  forceDark?: boolean;
}

/** Customer shell: top bar + content, no sider, no heavy antd. Own light/dark switch, dark by default; seats/checkout/tickets stay dark. Header has logo, uppercase nav, hamburger/avatar; mobile menu expands inline. No theater-select, search, or dead links. */
export const CustomerLayout = () => {
  const { t } = useTranslation();
  const themeMode = useAppStore((s) => s.theme);

  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  const matches = useMatches();
  const routeHandles = matches.map((match) => match.handle as RouteHandle | undefined);
  const fullBleed = routeHandles.some((handle) => handle?.fullBleed);
  const forceDark = routeHandles.some((handle) => handle?.forceDark);
  const isLight = themeMode === 'light' && !forceDark;

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    [
      'px-1 py-2 text-sm font-bold tracking-wide uppercase no-underline transition-colors duration-fast ease-out hover-fine:text-brand xl:text-base',
      isActive ? 'text-brand' : INK_80,
    ].join(' ');

  const mobileNavLinkClass = ({ isActive }: { isActive: boolean }) =>
    [
      'block py-3 text-sm font-bold tracking-wide uppercase no-underline transition-colors duration-fast ease-out hover-fine:text-brand',
      isActive ? 'text-brand' : INK_80,
    ].join(' ');

  return (
    <div
      className={[
        'cp-customer flex min-h-screen flex-col',
        'bg-(--cp-backdrop-base) bg-fixed text-[rgb(var(--cp-ink-rgb))] transition-colors duration-moderate ease-out',
        isLight ? 'cp-customer--light' : '',
      ]
        .filter(Boolean)
        .join(' ')}
      style={isLight ? undefined : { backgroundImage: cinemaGradient }}
    >
      {/* Real content in the shared `max-w-300` container aligned with <main> (header is just a full-bleed backdrop). */}
      <header className="sticky top-0 z-20 border-b border-(--cp-chrome-border) bg-(--cp-chrome-bg) backdrop-blur-md">
        <div className="mx-auto flex min-h-20 w-full max-w-300 items-center justify-between gap-4 px-4 sm:px-8">
          <Link
            to={PATHS.home}
            className={`group inline-flex items-center gap-2 text-xl font-bold tracking-tight no-underline ${INK}`}
          >
            <span
              className="text-2xl leading-none text-brand transition-transform duration-fast ease-out group-hover:scale-110"
              aria-hidden="true"
            >
              ◗
            </span>
            <span className="hidden sm:inline">{t('customer.brand')}</span>
          </Link>

          {/* Desktop nav (>=1024px); mobile unfolds inline below the header via hamburger. */}
          <nav className="hidden items-center gap-8 lg:flex">
            {NAV_ITEMS.map((item) => (
              <NavLink key={item.to} to={item.to} className={navLinkClass}>
                {t(item.labelKey)}
              </NavLink>
            ))}
          </nav>

          <div className="flex h-10 flex-none items-center gap-3">
            <button
              type="button"
              aria-label={t('customer.toggleMenu')}
              aria-expanded={mobileNavOpen}
              onClick={() => setMobileNavOpen((open) => !open)}
              className={`flex h-9.5 w-9.5 items-center justify-center lg:hidden ${INK}`}
            >
              <MenuIcon open={mobileNavOpen} />
            </button>

            <AvatarSwitcher />
          </div>
        </div>

        {mobileNavOpen ? (
          <nav
            className="border-t border-(--cp-chrome-border) lg:hidden"
            aria-label={t('customer.navMobile')}
          >
            <ul className="m-0 mx-auto flex max-w-300 list-none flex-col py-0 pr-4 pl-4 sm:pr-8 sm:pl-8">
              {NAV_ITEMS.map((item) => (
                <li key={item.to} className={`border-b ${INK_BORDER_10} last:border-0`}>
                  <NavLink
                    to={item.to}
                    className={mobileNavLinkClass}
                    onClick={() => setMobileNavOpen(false)}
                  >
                    {t(item.labelKey)}
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>
        ) : null}
      </header>

      <main
        className={
          fullBleed
            ? 'flex flex-1 flex-col'
            : 'mx-auto w-full max-w-300 flex-1 px-4 pt-2 pb-12 max-[900px]:pb-[calc(48px+var(--cp-bottomnav-height))]'
        }
      >
        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>
      </main>

      <Footer />

      <BottomNav />
    </div>
  );
};

export default CustomerLayout;
