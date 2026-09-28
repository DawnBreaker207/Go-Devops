import { useTranslation } from 'react-i18next';
import { INK_65 } from '@/theme/customerTw';

/** Placeholder for a future membership zone. */
export const MembershipTeaser = () => {
  const { t } = useTranslation();

  return (
    <section className="px-4 py-8 text-center">
      <span className="mb-2.5 inline-block rounded-full border border-[rgb(var(--cp-ink-rgb))]/18 bg-[rgb(var(--cp-ink-rgb))]/5 px-3.5 py-1 text-xs font-bold tracking-[0.6px] text-brand uppercase">
        {t('customer.accountComingSoon')}
      </span>
      <h2 className="mt-0 mb-1 text-base font-bold">{t('customer.accountMembership')}</h2>
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountMembershipHint')}</p>
    </section>
  );
};

export default MembershipTeaser;
