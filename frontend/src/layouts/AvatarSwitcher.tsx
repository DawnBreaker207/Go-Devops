import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';
import { useAppStore } from '@/stores/appStore';
import { useHasRole } from '@/hooks/useHasRole';
import { PATHS, accountTicketsPath } from '@/routes/paths';
import { surface, textOnBrand } from '@/theme';
import { FOCUS_RING, INK, INK_65, INK_85, INK_BORDER_18 } from '@/theme/customerTw';

const GaugeIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path d="M4 15a8 8 0 1 1 16 0" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    <path d="M12 15l4-5" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>
);

const ArrowRightIcon = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M5 12h14M13 6l6 6-6 6"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

const MoonIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z" />
  </svg>
);

const SunIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="1.7" />
    <path
      d="M12 2.5v2.5M12 19v2.5M2.5 12H5M19 12h2.5M5 5l1.8 1.8M17.2 17.2 19 19M19 5l-1.8 1.8M6.8 17.2 5 19"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
    />
  </svg>
);

const GlobeIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="12" r="8.5" stroke="currentColor" strokeWidth="1.6" />
    <path
      d="M3.5 12h17M12 3.5c2.5 2.3 2.5 15 0 17M12 3.5c-2.5 2.3-2.5 15 0 17"
      stroke="currentColor"
      strokeWidth="1.6"
    />
  </svg>
);

const PersonIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <circle cx="12" cy="8" r="3.5" stroke="currentColor" strokeWidth="1.7" />
    <path
      d="M4.5 20c1.4-4 4.2-6 7.5-6s6.1 2 7.5 6"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
    />
  </svg>
);

const TicketIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M3 8a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v2a1.5 1.5 0 0 0 0 3v2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-2a1.5 1.5 0 0 0 0-3V8Z"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinejoin="round"
    />
  </svg>
);

const LogoutIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

/** Avatar + header dropdown (no favorites page yet). Shared by both layouts (`variant` switches destinations); wrap with `.cp-customer` in MainLayout. Mobile uses a bottom sheet. */
export const AvatarSwitcher = ({ variant = 'customer' }: { variant?: 'customer' | 'operator' }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const logout = useAuthStore((s) => s.logout);
  const isCustomer = useHasRole('customer');
  const isAdmin = useHasRole('admin');

  const themeMode = useAppStore((s) => s.theme);
  const toggleTheme = useAppStore((s) => s.toggleTheme);
  const language = useAppStore((s) => s.language);
  const setLanguage = useAppStore((s) => s.setLanguage);

  const [open, setOpen] = useState(false);
  const [shown, setShown] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  // Set both states in one handler (no effect) so reopen replays the animation.
  const close = () => {
    setOpen(false);
    setShown(false);
  };

  useEffect(() => {
    if (!open) return;
    const raf = requestAnimationFrame(() => setShown(true));
    return () => cancelAnimationFrame(raf);
  }, [open]);

  // Outside click for desktop only; mobile has its own backdrop.
  useEffect(() => {
    if (!open) return;
    const onClick = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        close();
      }
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [open]);

  const initials = (user?.full_name ?? user?.email ?? 'U').charAt(0).toUpperCase();

  const handleLoginClick = () => {
    // Go straight to the login page (unlike mid-flow checkpoints that use a bottom sheet).
    close();
    navigate(variant === 'operator' ? PATHS.login : PATHS.customerLogin);
  };

  const handleLogout = () => {
    close();
    logout();
    navigate(PATHS.home, { replace: true });
  };

  return (
    <div className="relative" ref={rootRef}>
      {/* Preflight is off so `bg-transparent p-0` is required, or the button turns opaque. */}
      {/* Sub-44px hit target accepted per UI request. */}
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t('customer.accountMenu')}
        onClick={() => setOpen((o) => !o)}
        className={`flex items-center justify-center rounded-full border-2 border-solid bg-transparent p-0 transition-colors duration-fast ease-out hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6 ${INK_BORDER_18} ${FOCUS_RING}`}
      >
        {isAuthenticated ? (
          <span
            className="flex h-8 w-8 items-center justify-center rounded-full bg-brand text-[13px] font-bold"
            style={{ color: textOnBrand }}
          >
            {initials}
          </span>
        ) : (
          <span
            className={`flex h-8 w-8 items-center justify-center rounded-full text-sm ${INK_65}`}
          >
            👤
          </span>
        )}
      </button>

      {open ? (
        <>
          {/* Mobile-only backdrop; desktop already has outside-click. */}
          <div
            className="fixed inset-0 z-40 bg-black/50 transition-opacity duration-moderate ease-out md:hidden"
            style={{ opacity: shown ? 1 : 0 }}
            onClick={() => close()}
            aria-hidden="true"
          />

          <div
            role="menu"
            // Theme-aware background: avoids white-on-white text in both modes.
            style={{
              backgroundColor: themeMode === 'dark' ? surface.dark.base : surface.light.base,
            }}
            className={[
              'fixed inset-x-0 bottom-0 z-50 rounded-t-2xl border-t px-5 pt-3 pb-6 shadow-[0_-8px_32px_rgba(0,0,0,0.35)]',
              'transition-transform duration-moderate ease-out',
              shown ? 'translate-y-0' : 'translate-y-full',
              'md:absolute md:inset-x-auto md:top-full md:right-0 md:bottom-auto md:mt-2 md:w-72 md:rounded-xl md:border md:px-3 md:py-2 md:shadow-lg md:transition-opacity',
              'md:translate-y-0',
              open && !shown ? 'md:opacity-0' : 'md:opacity-100',
              INK_BORDER_18,
            ].join(' ')}
          >
            <div className="mx-auto mb-2 h-1 w-10 rounded-full bg-[rgb(var(--cp-ink-rgb))]/20 md:hidden" />

            <div className="flex items-center gap-3 px-1 py-2.5">
              {isAuthenticated ? (
                <span
                  className="flex h-11 w-11 flex-none items-center justify-center rounded-full bg-brand text-base font-bold"
                  style={{ color: textOnBrand }}
                >
                  {initials}
                </span>
              ) : (
                <span
                  className={`flex h-11 w-11 flex-none items-center justify-center rounded-full border border-dashed text-lg ${INK_BORDER_18} ${INK_65}`}
                >
                  👤
                </span>
              )}
              {isAuthenticated ? (
                <div className="min-w-0">
                  <div className={`truncate text-sm font-bold ${INK_85}`}>
                    {user?.full_name || user?.email}
                  </div>
                  {user?.full_name ? (
                    <div className={`truncate text-xs ${INK_65}`}>{user.email}</div>
                  ) : null}
                </div>
              ) : (
                <div className="min-w-0">
                  <div className={`text-sm font-bold ${INK_85}`}>{t('customer.guest')}</div>
                  <button
                    type="button"
                    onClick={handleLoginClick}
                    className="border-none bg-transparent p-0 text-xs font-semibold text-brand hover-fine:underline"
                  >
                    {t('customer.login')}
                  </button>
                </div>
              )}
            </div>

            <hr className={`my-1.5 border-t ${INK_BORDER_18}`} />

            {isAdmin ? (
              <>
                <Link
                  to={PATHS.dashboard}
                  onClick={() => close()}
                  className={`flex items-center justify-between gap-2 rounded-lg px-1 py-2.5 text-sm no-underline transition-colors duration-fast ease-out hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6 ${INK}`}
                >
                  <span className="flex items-center gap-2">
                    <GaugeIcon />
                    {t('customer.adminPanel')}
                  </span>
                  <ArrowRightIcon />
                </Link>
                <hr className={`my-1.5 border-t ${INK_BORDER_18}`} />
              </>
            ) : null}

            <div className="flex items-center justify-between gap-3 px-1 py-2.5">
              <span className={`flex items-center gap-2 text-sm font-medium ${INK_85}`}>
                <MoonIcon />
                {t('common.theme')}
              </span>
              {/* App switch: knob stays in flow, 2px shift when off. */}
              <button
                type="button"
                role="switch"
                aria-checked={themeMode === 'dark'}
                aria-label={t('customer.toggleTheme')}
                onClick={toggleTheme}
                className={`inline-flex h-7 w-12 min-w-12 shrink-0 cursor-pointer items-center rounded-full border border-transparent p-0 transition-colors duration-fast ease-out ${
                  themeMode === 'dark' ? 'bg-brand' : 'bg-[rgb(var(--cp-ink-rgb))]/20'
                }`}
              >
                <span
                  className={`pointer-events-none flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white shadow-md transition-transform duration-fast ease-out ${
                    themeMode === 'dark'
                      ? 'translate-x-5 text-slate-700'
                      : 'translate-x-0.5 text-amber-500'
                  }`}
                >
                  {themeMode === 'dark' ? <MoonIcon /> : <SunIcon />}
                </span>
              </button>
            </div>

            <div className="flex items-center justify-between gap-3 px-1 py-2.5">
              <span className={`flex items-center gap-2 text-sm font-medium ${INK_85}`}>
                <GlobeIcon />
                {t('common.language')}
              </span>
              <button
                type="button"
                role="switch"
                aria-checked={language === 'en'}
                aria-label={t('customer.toggleLanguage')}
                onClick={() => setLanguage(language === 'vi' ? 'en' : 'vi')}
                className={`inline-flex h-7 w-12 min-w-12 shrink-0 cursor-pointer items-center rounded-full border border-transparent p-0 transition-colors duration-fast ease-out ${
                  language === 'en' ? 'bg-brand' : 'bg-[rgb(var(--cp-ink-rgb))]/20'
                }`}
              >
                <span
                  className={`pointer-events-none flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-[10px] font-bold text-[#141414] shadow-md transition-transform duration-fast ease-out ${
                    language === 'en' ? 'translate-x-5' : 'translate-x-0.5'
                  }`}
                >
                  {language === 'vi' ? 'VI' : 'EN'}
                </span>
              </button>
            </div>

            {isAuthenticated ? (
              <>
                <hr className={`my-1.5 border-t ${INK_BORDER_18}`} />

                {isCustomer ? (
                  <>
                    <Link
                      to={PATHS.account}
                      onClick={() => close()}
                      className={`flex items-center gap-2 rounded-lg px-1 py-2.5 text-sm no-underline transition-colors duration-fast ease-out hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6 ${INK}`}
                    >
                      <PersonIcon />
                      {t('customer.myAccount')}
                    </Link>
                    <Link
                      to={accountTicketsPath()}
                      onClick={() => close()}
                      className={`flex items-center gap-2 rounded-lg px-1 py-2.5 text-sm no-underline transition-colors duration-fast ease-out hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6 ${INK}`}
                    >
                      <TicketIcon />
                      {t('customer.myTickets')}
                    </Link>
                  </>
                ) : (
                  // Staff/admin use `/profile` in both layouts.
                  <Link
                    to={PATHS.profile}
                    onClick={() => close()}
                    className={`flex items-center gap-2 rounded-lg px-1 py-2.5 text-sm no-underline transition-colors duration-fast ease-out hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6 ${INK}`}
                  >
                    <PersonIcon />
                    {t('common.profile')}
                  </Link>
                )}
                <button
                  type="button"
                  onClick={handleLogout}
                  className="flex w-full items-center gap-2 rounded-lg border-none bg-transparent px-1 py-2.5 text-left text-sm text-danger transition-colors duration-fast ease-out hover-fine:bg-danger/10"
                >
                  <LogoutIcon />
                  {t('common.logout')}
                </button>
              </>
            ) : null}
          </div>
        </>
      ) : null}
    </div>
  );
};

export default AvatarSwitcher;
