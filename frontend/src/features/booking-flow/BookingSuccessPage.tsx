import { useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import TicketCard from './components/TicketCard';
import { useOrderDetail } from './hooks/useOrders';
import { accountTicketsPath, PATHS } from '@/routes/paths';
import { useAuthStore } from '@/stores/authStore';
import { useBookingFlowStore } from '@/stores/bookingFlowStore';
import { errorMessage } from '@/utils/error';
import LinkButton from '@/components/ui/LinkButton';
import Notice from '@/components/ui/Notice';
import { INK_62 } from '@/theme/customerTw';

/** Post-payment landing on its own route. Re-reads the order (only proof of confirmation + tickets); clears the stale flow store (local-only) for a clean next run. */
export const BookingSuccessPage = () => {
  const { t } = useTranslation();
  const { bookingId } = useParams<{ bookingId: string }>();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const clear = useBookingFlowStore((s) => s.clear);
  const order = useOrderDetail(bookingId);

  // Run over: wipe the store (local-only, NO server cancel) for a clean next
  // run. The order renders from the URL bookingId, never the store.
  useEffect(() => {
    clear();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!isAuthenticated) {
    return (
      <Notice variant="info" role="status">
        <span>{t('customer.seatsNeedLogin')}</span>
      </Notice>
    );
  }
  if (!bookingId) {
    return <Notice variant="error">{errorMessage(undefined, t('common.somethingWrong'))}</Notice>;
  }
  if (order.error) {
    return <Notice variant="error">{errorMessage(order.error, t('common.somethingWrong'))}</Notice>;
  }
  if (order.isLoading) return <p className={INK_62}>{t('common.loading')}</p>;
  if (!order.data) return null;

  const confirmed = order.data.status === 'confirmed';

  return (
    <div className="mx-auto max-w-140 text-center">
      {confirmed ? (
        <span
          className="mb-3 inline-flex h-16 w-16 items-center justify-center rounded-full bg-brand text-[32px] leading-none font-bold text-on-brand"
          aria-hidden="true"
        >
          ✓
        </span>
      ) : null}

      <h1 className="mt-6 mb-4 text-center text-[28px] font-bold max-[640px]:text-[22px]">
        {confirmed ? t('customer.paymentSuccess') : t('customer.orderTitle')}
      </h1>

      <div className="text-left">
        {!confirmed ? (
          <Notice variant="info" role="status">
            {t('customer.notConfirmedYet', { status: t(`booking.status_${order.data.status}`) })}
          </Notice>
        ) : null}

        <TicketCard order={order.data} tickets={order.data.tickets} />
      </div>

      <div className="mt-5 flex gap-2.5">
        <LinkButton to={accountTicketsPath()} className="flex-1">
          {t('customer.myTickets')}
        </LinkButton>
        <LinkButton to={PATHS.home} variant="ghost" className="flex-1">
          {t('customer.backHome')}
        </LinkButton>
      </div>
    </div>
  );
};

export default BookingSuccessPage;
