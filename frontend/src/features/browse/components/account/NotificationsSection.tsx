import { useTranslation } from 'react-i18next';
import { useNotificationPreferences, useUpdateNotificationPreferences } from '@/hooks/useAccount';
import { errorMessage } from '@/utils/error';
import type { NotificationPreference } from '@/types';
import Notice from '@/components/ui/Notice';
import { INK_60, INK_65 } from '@/theme/customerTw';

interface ToggleRowProps {
  label: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  onToggle: () => void;
}

const ToggleRow = ({ label, description, checked, disabled, onToggle }: ToggleRowProps) => (
  <div className="flex items-center justify-between gap-4 border-b border-[rgb(var(--cp-ink-rgb))]/8 py-3 last:border-b-0">
    <div>
      <p className="text-sm font-semibold">{label}</p>
      <p className={`mt-0.5 text-xs ${INK_60}`}>{description}</p>
    </div>
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      className={[
        'relative h-6 w-11 shrink-0 cursor-pointer rounded-full border-none transition-colors duration-fast ease-out',
        "after:absolute after:top-0.75 after:left-0.75 after:h-4.5 after:w-4.5 after:rounded-full after:bg-white after:transition-transform after:duration-fast after:ease-out after:content-['']",
        'disabled:cursor-not-allowed disabled:opacity-55',
        checked ? 'bg-brand after:translate-x-5' : 'bg-[rgb(var(--cp-ink-rgb))]/22',
      ].join(' ')}
      disabled={disabled}
      onClick={onToggle}
    />
  </div>
);

/** Notification toggles for booking reminders and promos. */
export const NotificationsSection = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useNotificationPreferences();
  const update = useUpdateNotificationPreferences();

  const toggle = (key: keyof NotificationPreference) => {
    if (!data) return;
    update.mutate({ ...data, [key]: !data[key] });
  };

  return (
    <section>
      <h2 className="mt-0 mb-1 text-base font-bold">{t('customer.accountNotifications')}</h2>
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountNotificationsHint')}</p>

      {error ? (
        <Notice variant="error" className="mb-4">
          {errorMessage(error, t('common.somethingWrong'))}
        </Notice>
      ) : null}

      {isLoading ? <p className={INK_60}>{t('common.loading')}</p> : null}

      {data ? (
        <>
          <ToggleRow
            label={t('customer.accountNotifyBookingReminders')}
            description={t('customer.accountNotifyBookingRemindersDesc')}
            checked={data.booking_reminders}
            disabled={update.isPending}
            onToggle={() => toggle('booking_reminders')}
          />
          <ToggleRow
            label={t('customer.accountNotifyPromoOffers')}
            description={t('customer.accountNotifyPromoOffersDesc')}
            checked={data.promo_offers}
            disabled={update.isPending}
            onToggle={() => toggle('promo_offers')}
          />
        </>
      ) : null}
    </section>
  );
};

export default NotificationsSection;
