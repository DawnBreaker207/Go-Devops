import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import TicketCard from './components/TicketCard';
import { useMyOrders } from './hooks/useOrders';
import { checkoutPath, PATHS } from '@/routes/paths';
import { errorMessage } from '@/utils/error';
import './TicketCard.css';

/**
 * Ve cua toi. Figma frame My ticket (240-380): hai tab pill "Upcoming" /
 * "History" va mot luoi the ve.
 *
 * `GET /orders` la endpoint DUY NHAT trong nhanh /orders KHONG co
 * RequireRoles - no tra don cua chinh nguoi goi, nen role nao cung goi duoc.
 * No CO phan trang, va moi dong la `OrderStatusResponse` - tuc KHONG kem ve.
 * Ve chi co trong `GET /orders/:id`, nen the o day hien thong tin don va dan
 * sang trang chi tiet de xem ma ve.
 */

const PAGE_SIZE = 50;

type Tab = 'upcoming' | 'history';

export const MyTicketsPage = () => {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>('upcoming');
  const { data, isLoading, error } = useMyOrders({ page: 1, page_size: PAGE_SIZE });

  const orders = useMemo(() => data?.items ?? [], [data]);

  // Chia theo `showtime.ended` do BACKEND tu tinh, khong tu so sanh chuoi thoi
  // gian o client: cung mot thoi diem ve voi hai offset khac nhau tuy endpoint.
  const shown = useMemo(
    () =>
      orders.filter((order) =>
        tab === 'upcoming' ? !order.showtime?.ended : order.showtime?.ended
      ),
    [orders, tab]
  );

  return (
    <>
      <h1 className="cp-title" style={{ textAlign: 'center' }}>
        {t('customer.myTickets')}
      </h1>

      <div className="cp-tabs" role="tablist">
        {(['upcoming', 'history'] as const).map((key) => (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={tab === key}
            className={`cp-tab${tab === key ? ' cp-tab--active' : ''}`}
            onClick={() => setTab(key)}
          >
            {t(`customer.tab_${key}`)}
          </button>
        ))}
      </div>

      {error ? (
        <div className="cp-notice cp-notice--error" role="alert">
          {errorMessage(error, t('common.somethingWrong'))}
        </div>
      ) : null}

      {isLoading ? <p className="cp-muted">{t('common.loading')}</p> : null}

      {!isLoading && shown.length === 0 && !error ? (
        <div className="cp-empty">
          <p>{t('customer.noTickets')}</p>
          <Link to={PATHS.home} className="cp-btn cp-btn--primary">
            {t('customer.browseFilms')}
          </Link>
        </div>
      ) : null}

      <div className="cp-ticket-grid">
        {shown.map((order) => (
          <div key={order.id} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <TicketCard order={order} />
            <Link to={checkoutPath(order.id)} className="cp-btn cp-btn--ghost">
              {t('customer.viewOrder')}
            </Link>
          </div>
        ))}
      </div>
    </>
  );
};

export default MyTicketsPage;
