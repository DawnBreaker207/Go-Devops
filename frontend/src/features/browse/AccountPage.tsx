import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import DeleteAccountDialog from './components/account/DeleteAccountDialog';
import MembershipTeaser from './components/account/MembershipTeaser';
import NotificationsSection from './components/account/NotificationsSection';
import ProfileSection from './components/account/ProfileSection';
import SessionsSection from './components/account/SessionsSection';
import TicketsSection from './components/account/TicketsSection';
import TransactionsSection from './components/account/TransactionsSection';
import { useAuthStore } from '@/stores/authStore';
import { PATHS } from '@/routes/paths';
import Button from '@/components/ui/Button';
import LinkButton from '@/components/ui/LinkButton';
import EmptyState from '@/components/ui/EmptyState';
import StaticPage from '@/components/ui/StaticPage';
import {
  INK_65,
  INK_75,
  INK_BORDER_10,
  INK_BORDER_14,
  INK_BG_035,
  TRANSITION_FAST,
} from '@/theme/customerTw';

type Tab = 'profile' | 'tickets' | 'transactions' | 'notifications' | 'sessions' | 'membership';

const IdCardIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <rect x="2.5" y="5" width="19" height="14" rx="2" stroke="currentColor" strokeWidth="1.6" />
    <circle cx="8.5" cy="12" r="2.1" stroke="currentColor" strokeWidth="1.5" />
    <path
      d="M5.8 16.3c.5-1.5 1.6-2.3 2.7-2.3s2.2.8 2.7 2.3M14 10h5.5M14 13.5h5.5"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
    />
  </svg>
);

const HistoryIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path d="M3 12a9 9 0 1 0 3-6.7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    <path
      d="M3 4v4h4M12 8v4.5l3 2"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

const BellIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M6 10a6 6 0 1 1 12 0c0 4 1.5 5.5 1.5 5.5h-15S6 14 6 10Z"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinejoin="round"
    />
    <path
      d="M10 18.5a2 2 0 0 0 4 0"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
    />
  </svg>
);

const DevicesIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <rect x="2.5" y="4.5" width="13" height="9" rx="1.3" stroke="currentColor" strokeWidth="1.6" />
    <path d="M6 17h5.5" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    <rect x="16.5" y="9.5" width="5" height="9" rx="1" stroke="currentColor" strokeWidth="1.6" />
  </svg>
);

const CrownIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="m3 8 3.5 3L12 5l5.5 6L21 8l-1.6 9.5H4.6L3 8Z"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinejoin="round"
    />
  </svg>
);

const TicketIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M4 8.5a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v2a2 2 0 0 0 0 4v2a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-2a2 2 0 0 0 0-4v-2Z"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinejoin="round"
    />
    <path
      d="M13.5 6.5v2M13.5 11v2M13.5 15.5v2"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
    />
  </svg>
);

const TAB_ICON: Record<Tab, () => React.JSX.Element> = {
  profile: IdCardIcon,
  tickets: TicketIcon,
  transactions: HistoryIcon,
  notifications: BellIcon,
  sessions: DevicesIcon,
  membership: CrownIcon,
};

const TABS: Tab[] = [
  'profile',
  'tickets',
  'transactions',
  'notifications',
  'sessions',
  'membership',
];

const isTab = (value: string | null): value is Tab => (TABS as string[]).includes(value ?? '');

/** Customer account page with icon tabs sharing one card frame. */
export const AccountPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  const [searchParams, setSearchParams] = useSearchParams();
  // Tab lives in the URL so the "My tickets" link deep-opens the tickets tab. Read straight from the URL
  // (no state + effect) to stay within the hooks rules.
  const requestedTab = searchParams.get('tab');
  const tab: Tab = isTab(requestedTab) ? requestedTab : 'profile';
  const handleTab = (key: Tab) => {
    setSearchParams(key === 'profile' ? {} : { tab: key }, { replace: true });
  };
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  if (!user) {
    // No ProtectedRoute wrapper on this public route, so the signed-out state is handled here.
    return (
      <StaticPage>
        <EmptyState>{t('customer.accountNeedsLogin')}</EmptyState>
        <LinkButton to={PATHS.customerLogin} variant="primary">
          {t('customer.login')}
        </LinkButton>
      </StaticPage>
    );
  }

  const handleLogout = () => {
    logout();
    navigate(PATHS.home, { replace: true });
  };

  const handleDeleted = () => {
    setDeleteDialogOpen(false);
    logout();
    navigate(PATHS.home, { replace: true });
  };

  return (
    <div className="pb-6">
      <div className="mb-5">
        <h1 className="m-0 text-xl font-bold sm:text-2xl">{t('customer.accountTitle')}</h1>
        <p className={`mt-1 text-sm ${INK_65}`}>{t('customer.accountSubtitle')}</p>
      </div>

      <div className="mb-5 flex items-center gap-3.5">
        <div
          className="flex h-13 w-13 shrink-0 items-center justify-center rounded-full bg-brand text-xl font-bold text-on-brand"
          aria-hidden="true"
        >
          {user.full_name.trim().charAt(0).toUpperCase() || '?'}
        </div>
        <div className="min-w-0">
          <p className="m-0 truncate text-lg font-bold">{user.full_name}</p>
          <p className={`mt-0.5 truncate text-[13px] ${INK_65}`}>{user.email}</p>
        </div>
      </div>

      {/* Single card: icon tabs + content. */}
      <div
        className={`mb-6 flex flex-col overflow-hidden rounded-(--radius-card) border lg:flex-row ${INK_BORDER_14} ${INK_BG_035}`}
      >
        <div
          className={`flex flex-none gap-1 overflow-x-auto border-b p-2 lg:w-56 lg:flex-col lg:border-r lg:border-b-0 lg:p-3 ${INK_BORDER_10}`}
          role="tablist"
          aria-label={t('customer.accountTabsLabel')}
        >
          {TABS.map((key) => {
            const Icon = TAB_ICON[key];
            const active = tab === key;
            return (
              <button
                key={key}
                type="button"
                role="tab"
                aria-selected={active}
                className={[
                  'flex flex-none items-center gap-2.5 rounded-lg border-none bg-transparent px-3.5 py-2.5 text-left text-sm font-semibold whitespace-nowrap',
                  TRANSITION_FAST,
                  active
                    ? 'bg-brand/12 text-brand'
                    : `${INK_75} hover-fine:bg-[rgb(var(--cp-ink-rgb))]/6`,
                  key === 'membership' ? 'opacity-70' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
                onClick={() => handleTab(key)}
              >
                <Icon />
                {t(`customer.accountTab_${key}`)}
              </button>
            );
          })}
        </div>

        <div className="min-w-0 flex-1 p-4 sm:p-5">
          {tab === 'profile' ? <ProfileSection user={user} /> : null}
          {tab === 'tickets' ? <TicketsSection /> : null}
          {tab === 'transactions' ? <TransactionsSection /> : null}
          {tab === 'notifications' ? <NotificationsSection /> : null}
          {tab === 'sessions' ? <SessionsSection /> : null}
          {tab === 'membership' ? <MembershipTeaser /> : null}
        </div>
      </div>

      <section className={`rounded-(--radius-card) border p-5 ${INK_BORDER_14} ${INK_BG_035}`}>
        <h2 className="mt-0 mb-1 text-base font-bold">{t('common.logout')}</h2>
        <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountLogoutHint')}</p>
        <div className="mt-1 flex gap-2.5">
          <Button variant="ghost" onClick={handleLogout}>
            {t('common.logout')}
          </Button>
        </div>
      </section>

      <section className="mt-5 rounded-(--radius-card) border border-danger bg-[rgb(var(--cp-ink-rgb))]/4.5 p-5">
        <h2 className="mt-0 mb-1 text-base font-bold text-danger">
          {t('customer.accountDeleteTitle')}
        </h2>
        <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountDeleteHint')}</p>
        <div className="mt-1 flex gap-2.5">
          <Button variant="danger" onClick={() => setDeleteDialogOpen(true)}>
            {t('customer.accountDeleteCta')}
          </Button>
        </div>
      </section>

      {deleteDialogOpen ? (
        <DeleteAccountDialog onClose={() => setDeleteDialogOpen(false)} onDeleted={handleDeleted} />
      ) : null}
    </div>
  );
};

export default AccountPage;
