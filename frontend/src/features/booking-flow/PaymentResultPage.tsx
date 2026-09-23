import { useEffect, useRef } from 'react';
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useConfirmOrder, useOrderDetail, useOrderStatus } from './hooks/useOrders';
import { PATHS, bookingSuccessPath, selectSeatPath } from '@/routes/paths';
import { useAuthStore } from '@/stores/authStore';
import { errorMessage } from '@/utils/error';
import LinkButton from '@/components/ui/LinkButton';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { INK_62 } from '@/theme/customerTw';

/** Gateway result landing: query is display-only, decisions re-read the order (paid→confirm→success; confirmed→success; pending→wait+poll; settled→reason+rebook/home). */
export const PaymentResultPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const bookingId = searchParams.get('booking_id') ?? undefined;
  const detail = useOrderDetail(bookingId);
  const confirm = useConfirmOrder();

  const snap = detail.data;
  const paid = snap?.payment?.status === 'paid' || Boolean(snap?.paid_at);
  // IPN can lag the redirect: quiet-poll while pending-unpaid.
  const polling = snap?.status === 'pending' && !paid && !detail.error;
  const live = useOrderStatus(bookingId, polling);
  const livePaid = live.data?.payment?.status === 'paid' || Boolean(live.data?.paid_at);

  // Gateway says paid -> confirm at once (the old CheckoutStep auto-confirm,
  // now living here since checkout unloads).
  const confirmedRef = useRef(false);
  useEffect(() => {
    if (!bookingId || snap?.status !== 'pending' || (!paid && !livePaid)) return;
    if (confirmedRef.current || confirm.isPending) return;
    confirmedRef.current = true;
    confirm
      .mutateAsync(bookingId)
      .then(() => navigate(bookingSuccessPath(bookingId), { replace: true }))
      .catch(() => {
        confirmedRef.current = false;
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bookingId, snap?.status, paid, livePaid]);

  if (!isAuthenticated) {
    return (
      <Notice variant="info" role="status">
        <span>{t('customer.seatsNeedLogin')}</span>
      </Notice>
    );
  }
  if (!bookingId) {
    return (
      <div className="mx-auto max-w-140 text-center">
        <Notice variant="error">{t('common.somethingWrong')}</Notice>
        <div className="mt-5">
          <LinkButton to={PATHS.home} variant="ghost">
            {t('customer.backHome')}
          </LinkButton>
        </div>
      </div>
    );
  }
  if (detail.error) {
    return (
      <div className="mx-auto max-w-140 text-center">
        <Notice variant="error">{errorMessage(detail.error, t('common.somethingWrong'))}</Notice>
        <div className="mt-5 flex gap-2.5">
          <Button
            variant="ghost"
            block
            onClick={() => void detail.refetch()}
            disabled={detail.isFetching}
          >
            {t('common.retry')}
          </Button>
          <LinkButton to={PATHS.home} variant="ghost" className="flex-1">
            {t('customer.backHome')}
          </LinkButton>
        </div>
      </div>
    );
  }
  if (detail.isLoading || !snap) return <p className={INK_62}>{t('common.loading')}</p>;
  if (snap.status === 'confirmed') {
    return <Navigate to={bookingSuccessPath(snap.id)} replace />;
  }

  const settled = snap.status === 'expired' || snap.status === 'refunded';
  const failed = snap.payment?.status === 'failed';
  const confirming = confirm.isPending || ((paid || livePaid) && !confirm.error);

  return (
    <div className="mx-auto max-w-140 text-center">
      <h1 className="mt-6 mb-4 text-center text-[28px] font-bold max-[640px]:text-[22px]">
        {t('customer.paymentResultTitle')}
      </h1>

      <div className="text-left">
        {settled ? (
          <Notice variant="error">
            {t('customer.orderSettled', {
              status: t(`booking.status_${snap.status}`),
              reason: snap.status_reason
                ? t(`booking.reason_${snap.status_reason}`, snap.status_reason)
                : t('customer.payReasonUnknown'),
            })}
          </Notice>
        ) : failed ? (
          <Notice variant="error">
            {t('customer.paymentFailed', {
              reason: snap.payment?.status_reason
                ? t(`customer.payReason_${snap.payment.status_reason}`, snap.payment.status_reason)
                : t('customer.payReasonUnknown'),
            })}
          </Notice>
        ) : (
          <Notice variant="info" role="status">
            {confirm.error
              ? errorMessage(confirm.error, t('common.somethingWrong'))
              : t('customer.paymentReturnWaiting')}
          </Notice>
        )}
      </div>

      {confirming && !confirm.error ? <p className={INK_62}>{t('common.loading')}</p> : null}

      <div className="mt-5 flex gap-2.5">
        {/* Same tab, so the flow store survives - back opens the same order. */}
        <Button block onClick={() => navigate(selectSeatPath(snap.showtime_id))}>
          {t('customer.paymentReturnBack')}
        </Button>
        <LinkButton to={PATHS.home} variant="ghost" className="flex-1">
          {t('customer.backHome')}
        </LinkButton>
      </div>
    </div>
  );
};

export default PaymentResultPage;
