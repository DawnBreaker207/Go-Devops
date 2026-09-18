import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { formatCountdown, useCountdown } from './useCountdown';
import {
  useCancelOrder,
  useConfirmOrder,
  useOrderDetail,
  useOrderStatus,
  usePayOrder,
} from './hooks/useOrders';
import { PATHS, orderSuccessPath } from '@/routes/paths';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';
import './CheckoutPage.css';

/**
 * Trang thanh toan cua mot don dang giu cho.
 *
 * Luong that cua backend: `POST /orders/:id/pay` tra ve `redirect_url` cua cong
 * thanh toan, cong do goi IPN ve server, roi client goi
 * `POST /orders/:id/confirm` de chot.
 *
 * Cong duoc mo o TAB MOI, khong dieu huong ca trang, vi `payment.return_redirect_url`
 * dang de RONG - nghia la trang return cua backend tra ve JSON tho chu khong
 * dua nguoi dung ve lai giao dien. Mo tab moi giu trang nay song de con dem
 * nguoc va chot don.
 *
 * KHONG bao gio tin mot "da thanh toan" tu phia client: trang thai that chi den
 * tu `GET /orders/:id/status`, va `confirm` van se tu choi neu cong chua bao.
 */
export const CheckoutPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { bookingId } = useParams<{ bookingId: string }>();

  const [actionError, setActionError] = useState<string | null>(null);
  const [gatewayOpened, setGatewayOpened] = useState(false);

  const order = useOrderDetail(bookingId);
  const pay = usePayOrder();
  const confirm = useConfirmOrder();
  const cancel = useCancelOrder();

  const isPending = order.data?.status === 'pending';
  // Chi do khi da mo cong: truoc do khong co gi de doi.
  const status = useOrderStatus(bookingId, isPending && gatewayOpened);
  const paid = status.data?.payment?.status === 'paid' || Boolean(status.data?.paid_at);

  const secondsLeft = useCountdown(order.data?.expires_at);
  const expired = order.data?.status === 'expired' || (isPending && secondsLeft === 0);

  // Cong bao da tra tien -> chot don ngay, khong bat khach bam them mot nut nua.
  useEffect(() => {
    if (!bookingId || !paid || !isPending || confirm.isPending) return;
    confirm
      .mutateAsync(bookingId)
      .then(() => navigate(orderSuccessPath(bookingId), { replace: true }))
      .catch((error) => setActionError(errorMessage(error, t('common.somethingWrong'))));
  }, [bookingId, paid, isPending, confirm, navigate, t]);

  const handlePay = async () => {
    if (!bookingId) return;
    setActionError(null);
    try {
      // Bo trong `provider` de backend dung cong mac dinh cua no.
      const result = await pay.mutateAsync({ bookingId, payload: {} });
      setGatewayOpened(true);
      window.open(result.redirect_url, '_blank', 'noopener');
    } catch (error) {
      setActionError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleCancel = async () => {
    if (!bookingId) return;
    setActionError(null);
    try {
      await cancel.mutateAsync(bookingId);
      navigate(PATHS.home);
    } catch (error) {
      setActionError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  if (order.error) {
    return (
      <div className="cp-notice cp-notice--error" role="alert">
        {errorMessage(order.error, t('common.somethingWrong'))}
      </div>
    );
  }
  if (order.isLoading) return <p className="cp-muted">{t('common.loading')}</p>;
  if (!order.data) return null;

  const o = order.data;
  const countdownClass = expired
    ? ' cp-countdown--over'
    : secondsLeft < 120
      ? ' cp-countdown--urgent'
      : '';

  return (
    <div className="cp-checkout">
      <h1 className="cp-title">{t('customer.orderTitle')}</h1>

      {isPending ? (
        <div className={`cp-countdown${countdownClass}`}>
          <span>{expired ? t('customer.holdExpired') : t('customer.holdLeft')}</span>
          <span className="cp-countdown__time tabular-nums">{formatCountdown(secondsLeft)}</span>
        </div>
      ) : null}

      {actionError ? (
        <div className="cp-notice cp-notice--error" role="alert">
          {actionError}
        </div>
      ) : null}

      <section className="cp-checkout__section">
        <h2 className="cp-checkout__section-title">{t('customer.schedule')}</h2>
        <div className="cp-kv">
          <span className="cp-kv__label">{t('report.movie')}</span>
          <span className="cp-kv__value">{o.showtime?.movie_title ?? '-'}</span>
        </div>
        <div className="cp-kv">
          <span className="cp-kv__label">{t('report.hall')}</span>
          <span className="cp-kv__value">{o.showtime?.hall_name ?? '-'}</span>
        </div>
        <div className="cp-kv">
          <span className="cp-kv__label">{t('report.startAt')}</span>
          <span className="cp-kv__value">{formatDateTime(o.showtime?.start_at)}</span>
        </div>
      </section>

      <section className="cp-checkout__section">
        <h2 className="cp-checkout__section-title">{t('customer.transaction')}</h2>
        {/* Ve chi ton tai sau khi don duoc chot; don dang giu cho thi chua co. */}
        {o.tickets.length > 0 ? (
          o.tickets.map((ticket) => (
            <div key={ticket.id} className="cp-kv">
              <span className="cp-kv__label">
                {ticket.seat_label} · {t(`hall.seatType_${ticket.seat_type}`)}
              </span>
              <span className="cp-kv__value tabular-nums">{formatVND(ticket.price)}</span>
            </div>
          ))
        ) : (
          <div className="cp-kv">
            <span className="cp-kv__label">{t('customer.seatsHeld')}</span>
            <span className="cp-kv__value">{t('customer.awaitingPayment')}</span>
          </div>
        )}

        <div className="cp-checkout__divider" />
        <div className="cp-checkout__total">
          <span>{t('customer.totalPayment')}</span>
          <span className="tabular-nums">{formatVND(o.total_amount)}</span>
        </div>
      </section>

      <div className="cp-checkout__actions">
        {isPending && !expired ? (
          <>
            <button
              type="button"
              className="cp-btn cp-btn--primary cp-btn--block"
              onClick={() => void handlePay()}
              disabled={pay.isPending || confirm.isPending}
            >
              {pay.isPending ? t('common.loading') : t('customer.payNow')}
            </button>
            <button
              type="button"
              className="cp-btn cp-btn--ghost cp-btn--block"
              onClick={() => void handleCancel()}
              disabled={cancel.isPending}
            >
              {t('customer.cancelOrder')}
            </button>
          </>
        ) : null}

        {expired ? (
          <button
            type="button"
            className="cp-btn cp-btn--ghost cp-btn--block"
            onClick={() => navigate(PATHS.home)}
          >
            {t('customer.backHome')}
          </button>
        ) : null}

        {o.status === 'confirmed' && bookingId ? (
          <button
            type="button"
            className="cp-btn cp-btn--primary cp-btn--block"
            onClick={() => navigate(orderSuccessPath(bookingId))}
          >
            {t('customer.viewTickets')}
          </button>
        ) : null}
      </div>

      {gatewayOpened && isPending && !expired ? (
        <p className="cp-fineprint">{t('customer.gatewayHint')}</p>
      ) : (
        <p className="cp-fineprint">{t('customer.holdHint')}</p>
      )}
    </div>
  );
};

export default CheckoutPage;
