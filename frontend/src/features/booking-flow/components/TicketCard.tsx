import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useTicketQR } from '../hooks/useTicketQR';
import type { OrderStatus, Ticket } from '@/types';
import { filmPath, PATHS } from '@/routes/paths';
import { formatDateTime, formatVND } from '@/utils/format';
import Button from '@/components/ui/Button';
import LinkButton from '@/components/ui/LinkButton';
import Notice from '@/components/ui/Notice';
import {
  HOVER_INK_BORDER,
  INK_62,
  INK_65,
  INK_85,
  INK_BG_03,
  INK_BORDER_20,
  INK_BORDER_28,
} from '@/theme/customerTw';

interface TicketCardProps {
  order: OrderStatus;
  /** `undefined` = tickets not loaded; `[]` = loaded and truly none. Render differently: "unloaded" as "none" prints a lying "Tickets (0)" on confirmed orders. */
  tickets?: Ticket[];
  /** "Rebook quickly" button - "Past" group of the My-tickets tab only. */
  showRebook?: boolean;
}

/** Legacy `.cp-ticket__label` - small label above the value, theme-reactive. */
const FIELD_LABEL_CLASS = `mb-0.5 block text-xs ${INK_65}`;
const FIELD_VALUE_CLASS = 'text-base font-semibold';

const STATUS_BADGE_CLASS: Record<OrderStatus['status'], string> = {
  pending: 'bg-warning text-[#1a1200]',
  confirmed: 'bg-brand text-on-brand',
  expired: `bg-[rgb(var(--cp-ink-rgb))]/18 ${INK_85}`,
  refunded: `bg-[rgb(var(--cp-ink-rgb))]/18 ${INK_85}`,
};

/** One ticket's QR: offline cache FIRST (instant if present), then silent `GET /tickets/:id/qr` refresh. Offline still shows the cached QR - never block UI on network. Tap to zoom (gates need it big) - closes on backdrop, Esc, or Back. */
const TicketQRImage = ({ ticket }: { ticket: Ticket }) => {
  const { t } = useTranslation();
  const { qrBase64, loading } = useTicketQR(ticket.id);
  const [zoomed, setZoomed] = useState(false);

  useEffect(() => {
    if (!zoomed) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setZoomed(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [zoomed]);

  const src = qrBase64 ? `data:image/png;base64,${qrBase64}` : null;
  const alt = t('customer.ticketQrAlt', { seat: ticket.seat_label });

  return (
    <div className="flex flex-col items-center gap-1">
      {src ? (
        <button
          type="button"
          onClick={() => setZoomed(true)}
          aria-label={t('customer.ticketQrZoom', { seat: ticket.seat_label })}
          title={t('customer.ticketQrZoom', { seat: ticket.seat_label })}
          className="cursor-zoom-in border-none bg-transparent p-0 transition-transform duration-fast ease-out hover-fine:scale-105 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand"
        >
          <img className="block h-22 w-22 rounded-md bg-white p-1" src={src} alt={alt} />
        </button>
      ) : (
        <div
          className={`flex h-22 w-22 items-center justify-center rounded-md border border-dashed text-center text-[11px] ${INK_BORDER_28} ${INK_65}`}
          aria-hidden="true"
        >
          {loading ? t('common.loading') : '-'}
        </div>
      )}
      <span className={`font-mono text-[13px] tracking-[1px] break-all ${INK_85}`}>
        {ticket.seat_label}
      </span>
      {zoomed && src ? (
        <div
          className="fixed inset-0 z-1000 flex items-center justify-center bg-black/70 p-6"
          role="presentation"
          onClick={() => setZoomed(false)}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label={alt}
            onClick={(event) => event.stopPropagation()}
            className="flex flex-col items-center gap-3 rounded-2xl bg-white p-5"
          >
            <img src={src} alt={alt} className="block h-auto w-[min(72vw,300px)]" />
            <span className="font-mono text-sm font-bold tracking-[1px] text-[#141414]">
              {ticket.seat_label}
            </span>
            <Button variant="ghost" onClick={() => setZoomed(false)}>
              {t('customer.back')}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
};

/** One order as a ticket card: date, movie, count + time, then a button. Theme-reactive; QR stays white for scanners. */
export const TicketCard = ({ order, tickets, showRebook }: TicketCardProps) => {
  const { t } = useTranslation();
  const loaded = tickets !== undefined;
  const seats = (tickets ?? []).map((ticket) => ticket.seat_label).join(', ');

  // Venue-cancelled showtime (POST /admin/showtimes/:id/cancel): order goes
  // `refunded` with its own `showtime_cancelled` reason - never lumped with
  // used tickets or customer-cancelled/expired orders.
  const showtimeCancelled =
    order.status === 'refunded' && order.status_reason === 'showtime_cancelled';

  const movieId = order.showtime?.movie_id;

  return (
    <article
      className={[
        `relative flex flex-col gap-3.5 rounded-xl border p-4.5 ${INK_BORDER_20} ${INK_BG_03}`,
        `transition-colors duration-fast ease-out ${HOVER_INK_BORDER}`,
        "after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:shadow-[0_16px_28px_rgba(0,0,0,0.35)] after:transition-opacity after:duration-base after:ease-out after:content-['']",
        '[@media(hover:hover)_and_(pointer:fine)]:group-hover:after:opacity-100',
      ].join(' ')}
    >
      {showtimeCancelled ? (
        <span className="self-start rounded-full bg-danger px-2.5 py-0.5 text-[11px] font-bold text-white">
          {t('customer.showtimeCancelledBadge')}
        </span>
      ) : (
        <span
          className={`self-start rounded-full px-2.5 py-0.5 text-[11px] font-bold ${STATUS_BADGE_CLASS[order.status]}`}
        >
          {t(`booking.status_${order.status}`)}
        </span>
      )}

      <div>
        <span className={FIELD_LABEL_CLASS}>{t('report.date')}</span>
        <span className={FIELD_VALUE_CLASS}>{formatDateTime(order.showtime?.start_at)}</span>
      </div>

      <div>
        <span className={FIELD_LABEL_CLASS}>{t('report.movie')}</span>
        <span className="text-lg leading-[1.3] font-bold uppercase">
          {order.showtime?.movie_title ?? '-'}
        </span>
      </div>

      <div className="flex justify-between gap-4">
        <div>
          <span className={FIELD_LABEL_CLASS}>
            {loaded ? t('customer.ticketCount', { count: tickets.length }) : t('customer.seats')}
          </span>
          <span className={FIELD_VALUE_CLASS}>
            {seats
              ? seats
              : order.status === 'pending'
                ? /* Live hold - still payable. */
                  t('customer.awaitingPayment')
                : /* Dead order: empty because tickets never issue, NOT because
                     "awaiting payment" - the reason sits in status_reason below. */
                  '—'}
            {!seats && !loaded ? ` · ${t('customer.openOrderForSeats')}` : null}
          </span>
        </div>
        <div className="text-right">
          <span className={FIELD_LABEL_CLASS}>{t('customer.totalPayment')}</span>
          <span className={`${FIELD_VALUE_CLASS} tabular-nums`}>
            {formatVND(order.total_amount)}
          </span>
        </div>
      </div>

      <div>
        <span className={FIELD_LABEL_CLASS}>{t('report.hall')}</span>
        <span className={FIELD_VALUE_CLASS}>{order.showtime?.hall_name ?? '-'}</span>
      </div>

      {/* `status_reason` only exists on expired/refunded orders - show the WHY, never swallow it. */}
      {order.status_reason && !showtimeCancelled ? (
        <p className={`m-0 text-[13px] ${INK_62}`}>
          {t(`booking.reason_${order.status_reason}`, order.status_reason)}
        </p>
      ) : null}

      {tickets && tickets.length > 0 ? (
        <>
          {/* Perforation: dashed tear + two notches marking the scannable stub. Notches use `--cp-canvas` (true page bg, mode-reactive). */}
          <div
            aria-hidden="true"
            className={[
              'relative -mx-4.5 h-0 border-t-2 border-dashed border-[rgb(var(--cp-ink-rgb))]/22',
              "before:absolute before:top-0 before:-left-2 before:h-4 before:w-4 before:-translate-y-1/2 before:rounded-full before:bg-(--cp-canvas) before:content-['']",
              "after:absolute after:top-0 after:-right-2 after:h-4 after:w-4 after:-translate-y-1/2 after:rounded-full after:bg-(--cp-canvas) after:content-['']",
            ].join(' ')}
          />
          <div>
            <span className={FIELD_LABEL_CLASS}>{t('customer.ticketCodes')}</span>
            <div className="mt-1.5 grid grid-cols-[repeat(auto-fill,minmax(88px,1fr))] gap-3">
              {tickets.map((ticket) => (
                <TicketQRImage key={ticket.id} ticket={ticket} />
              ))}
            </div>
          </div>
        </>
      ) : null}

      {showRebook ? (
        movieId ? (
          <LinkButton to={filmPath(movieId)} variant="ghost" block>
            {t('customer.rebookQuick')}
          </LinkButton>
        ) : (
          <Notice variant="info" role="status" stack>
            <p className="m-0">{t('customer.movieRemovedNotice')}</p>
            <LinkButton to={PATHS.home} variant="ghost">
              {t('customer.backHome')}
            </LinkButton>
          </Notice>
        )
      ) : null}
    </article>
  );
};

export default TicketCard;
