import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import TicketCard from './components/TicketCard';
import { useOrderDetail } from './hooks/useOrders';
import { PATHS } from '@/routes/paths';
import { errorMessage } from '@/utils/error';

/**
 * Man sau khi thanh toan thanh cong. Figma frame Payment Success (122-1100).
 *
 * Doc lai don tu server chu khong tin state truyen qua router: chi `GET
 * /orders/:id` moi biet don da thuc su chot va ve da phat hay chua.
 */
export const OrderSuccessPage = () => {
  const { t } = useTranslation();
  const { bookingId } = useParams<{ bookingId: string }>();
  const order = useOrderDetail(bookingId);

  if (order.error) {
    return (
      <div className="cp-notice cp-notice--error" role="alert">
        {errorMessage(order.error, t('common.somethingWrong'))}
      </div>
    );
  }
  if (order.isLoading) return <p className="cp-muted">{t('common.loading')}</p>;
  if (!order.data) return null;

  const confirmed = order.data.status === 'confirmed';

  return (
    <div style={{ maxWidth: 560, margin: '0 auto' }}>
      <h1 className="cp-title" style={{ textAlign: 'center' }}>
        {confirmed ? t('customer.paymentSuccess') : t('customer.orderTitle')}
      </h1>

      {!confirmed ? (
        <div className="cp-notice cp-notice--info">
          {t('customer.notConfirmedYet', { status: t(`booking.status_${order.data.status}`) })}
        </div>
      ) : null}

      <TicketCard order={order.data} tickets={order.data.tickets} />

      <div style={{ display: 'flex', gap: 10, marginTop: 20 }}>
        <Link to={PATHS.myTickets} className="cp-btn cp-btn--primary" style={{ flex: 1 }}>
          {t('customer.myTickets')}
        </Link>
        <Link to={PATHS.home} className="cp-btn cp-btn--ghost" style={{ flex: 1 }}>
          {t('customer.backHome')}
        </Link>
      </div>
    </div>
  );
};

export default OrderSuccessPage;
