import { useTranslation } from 'react-i18next';
import { useRevokeSession, useSessions } from '@/hooks/useAccount';
import { errorMessage } from '@/utils/error';
import { formatDateTime } from '@/utils/format';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import { INK_60, INK_65 } from '@/theme/customerTw';

/** Device session list with per-device sign-out. */
export const SessionsSection = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useSessions();
  const revoke = useRevokeSession();

  return (
    <section>
      <h2 className="mt-0 mb-1 text-base font-bold">{t('customer.accountSessions')}</h2>
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountSessionsHint')}</p>

      {error ? (
        <Notice variant="error" className="mb-4">
          {errorMessage(error, t('common.somethingWrong'))}
        </Notice>
      ) : null}

      {isLoading ? <p className={INK_60}>{t('common.loading')}</p> : null}

      {!isLoading && data && data.length === 0 && !error ? (
        <EmptyState>{t('customer.accountNoSessions')}</EmptyState>
      ) : null}

      <div className="flex flex-col gap-2.5">
        {data?.map((session) => (
          <div
            className="flex items-center justify-between gap-3 rounded-[10px] bg-[rgb(var(--cp-ink-rgb))]/5 px-3.5 py-3 max-[640px]:flex-col max-[640px]:items-start"
            key={session.id}
          >
            <div>
              <p className="m-0 max-w-80 truncate text-[13px] font-semibold">
                {session.user_agent || t('customer.accountUnknownDevice')}
                {session.is_current ? (
                  <span className="ml-2 inline-block rounded-full bg-brand-soft px-2 py-px text-[11px] font-bold text-brand-active">
                    {t('customer.accountCurrentDevice')}
                  </span>
                ) : null}
              </p>
              <p className={`mt-0.75 text-xs ${INK_60}`}>
                {t('customer.accountSessionLastUsed')}:{' '}
                {session.last_used_at ? formatDateTime(session.last_used_at) : '-'}
              </p>
            </div>
            {!session.is_current ? (
              <Button
                variant="ghost"
                disabled={revoke.isPending}
                onClick={() => revoke.mutate(session.id)}
              >
                {t('customer.accountSignOutDevice')}
              </Button>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
};

export default SessionsSection;
