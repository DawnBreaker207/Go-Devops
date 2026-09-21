import { useTranslation } from 'react-i18next';
import type { OrderStatus, Ticket } from '@/types';
import { formatDateTime, formatVND } from '@/utils/format';
import '../TicketCard.css';

interface TicketCardProps {
  order: OrderStatus;
  /**
   * `undefined` = KHONG duoc nap ve (danh sach `GET /orders` tra
   * OrderStatusResponse, khong kem ve). `[]` = da nap va don that su khong co
   * ve nao (don dang giu cho).
   *
   * Hai truong hop nay phai ve khac nhau: coi "chua nap" thanh "khong co ve"
   * lam mot don DA XAC NHAN hien ra "Ve (0) - Cho thanh toan", tuc noi sai.
   */
  tickets?: Ticket[];
  /** "Rebook quickly" button - "Past" group of the My-tickets tab only. */
  showRebook?: boolean;
}

/**
 * Mot don hien duoi dang the ve. Figma frame My ticket (240-380): Date, Movie
 * Title, Ticket (n) + gio, roi mot nut o duoi.
 */
export const TicketCard = ({ order, tickets }: TicketCardProps) => {
  const { t } = useTranslation();
  const loaded = tickets !== undefined;
  const seats = (tickets ?? []).map((ticket) => ticket.seat_label).join(', ');

  return (
    <article className="cp-ticket">
      <span className={`cp-ticket__status cp-ticket__status--${order.status}`}>
        {t(`booking.status_${order.status}`)}
      </span>

      <div>
        <span className="cp-ticket__label">{t('report.date')}</span>
        <span className="cp-ticket__value">{formatDateTime(order.showtime?.start_at)}</span>
      </div>

      <div>
        <span className="cp-ticket__label">{t('report.movie')}</span>
        <span className="cp-ticket__movie">{order.showtime?.movie_title ?? '-'}</span>
      </div>

      <div className="cp-ticket__row">
        <div>
          <span className="cp-ticket__label">
            {loaded ? t('customer.ticketCount', { count: tickets.length }) : t('customer.seats')}
          </span>
          <span className="cp-ticket__value">
            {seats
              ? seats
              : loaded
                ? t('customer.awaitingPayment')
                : /* Danh sach don khong kem ve; mo don ra moi co. */
                  t('customer.openOrderForSeats')}
          </span>
        </div>
        <div style={{ textAlign: 'right' }}>
          <span className="cp-ticket__label">{t('customer.totalPayment')}</span>
          <span className="cp-ticket__value tabular-nums">{formatVND(order.total_amount)}</span>
        </div>
      </div>

      <div>
        <span className="cp-ticket__label">{t('report.hall')}</span>
        <span className="cp-ticket__value">{order.showtime?.hall_name ?? '-'}</span>
      </div>

      {/* `status_reason` chi co tren don da het han / hoan tien, va no giai
          thich VI SAO - khong nuot di. */}
      {order.status_reason ? (
        <p className="cp-muted" style={{ margin: 0, fontSize: 13 }}>
          {t(`booking.reason_${order.status_reason}`, order.status_reason)}
        </p>
      ) : null}

      {tickets && tickets.length > 0 ? (
        <div>
          <span className="cp-ticket__label">{t('customer.ticketCodes')}</span>
          {tickets.map((ticket) => (
            <div key={ticket.id} className="cp-ticket__code">
              {ticket.seat_label} — {ticket.code}
            </div>
          ))}
        </div>
      ) : null}
    </article>
  );
};

export default TicketCard;
