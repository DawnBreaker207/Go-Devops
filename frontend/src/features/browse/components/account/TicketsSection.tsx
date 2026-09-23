import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import TicketCard from '@/features/booking-flow/components/TicketCard';
import { useMyOrders, useOrderDetail } from '@/features/booking-flow/hooks/useOrders';
import { PATHS } from '@/routes/paths';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND, toCinemaTime } from '@/utils/format';
import type { BookingStatus, OrderStatus } from '@/types';
import Button from '@/components/ui/Button';
import LinkButton from '@/components/ui/LinkButton';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import {
  INK,
  INK_60,
  INK_62,
  INK_65,
  INK_75,
  INK_BG_035,
  INK_BORDER_14,
  INK_BORDER_28,
} from '@/theme/customerTw';

const PAGE_SIZE = 50;
const HISTORY_MONTHS = 3;

type InnerTab = 'upcoming' | 'history';

/** Cancelled showtimes group with past orders. */
const isShowtimeCancelled = (order: OrderStatus) =>
  order.status === 'refunded' && order.status_reason === 'showtime_cancelled';

const isHistoryOrder = (order: OrderStatus) =>
  Boolean(order.showtime?.ended) || isShowtimeCancelled(order);

const PILL_CLASS = (active: boolean) =>
  [
    'cursor-pointer rounded-full border px-4.5 py-1.5 text-[13px] transition-colors',
    'duration-fast ease-out focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand',
    active
      ? 'border-transparent bg-brand font-bold text-on-brand'
      : `bg-transparent ${INK_BORDER_28} ${INK_75}`,
  ].join(' ');

const ROW_BADGE_CLASS: Record<BookingStatus, string> = {
  pending: 'bg-warning text-[#1a1200]',
  confirmed: 'bg-brand text-on-brand',
  expired: `bg-[rgb(var(--cp-ink-rgb))]/18 ${INK_75}`,
  refunded: `bg-[rgb(var(--cp-ink-rgb))]/18 ${INK_75}`,
};

/** Ticket list with upcoming and past tabs plus per-order detail. */
export const TicketsSection = () => {
  const { t } = useTranslation();
  const [tab, setTab] = useState<InnerTab>('upcoming');
  const [openedId, setOpenedId] = useState<string | null>(null);
  // "Show more" expands the Past group beyond the default 3-month window.
  const [showAllHistory, setShowAllHistory] = useState(false);
  const { data, isLoading, error } = useMyOrders({ page: 1, page_size: PAGE_SIZE });

  const orders = useMemo(
    () => (data?.items ?? []).filter((order) => order.status === 'confirmed'),
    [data]
  );

  const upcoming = useMemo(() => orders.filter((order) => !isHistoryOrder(order)), [orders]);
  const history = useMemo(() => orders.filter((order) => isHistoryOrder(order)), [orders]);

  const historyCutoff = useMemo(
    () => toCinemaTime(new Date()).subtract(HISTORY_MONTHS, 'month'),
    []
  );
  // `GET /orders` has no from/to (only `GET /admin/orders` does), so filter client-side;
  // prefer showtime, fall back to creation date for records without one.
  const recentHistory = useMemo(
    () =>
      history.filter((order) => {
        const at = order.showtime?.start_at ?? order.created_at;
        return toCinemaTime(at).isAfter(historyCutoff);
      }),
    [history, historyCutoff]
  );
  const hasOlderHistory = history.length > recentHistory.length;

  const historyShown = showAllHistory ? history : recentHistory;
  const shown = tab === 'upcoming' ? upcoming : historyShown;
  const opened = openedId ? (orders.find((order) => order.id === openedId) ?? null) : null;

  return (
    <section>
      <h2 className="mt-0 mb-1 text-base font-bold">{t('customer.myTickets')}</h2>
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountTicketsHint')}</p>

      {opened ? (
        <TicketDetail
          order={opened}
          isHistory={tab === 'history'}
          onBack={() => setOpenedId(null)}
        />
      ) : (
        <>
          <div className="mb-5 flex gap-2" role="tablist">
            {(['upcoming', 'history'] as const).map((key) => (
              <button
                key={key}
                type="button"
                role="tab"
                aria-selected={tab === key}
                className={PILL_CLASS(tab === key)}
                onClick={() => setTab(key)}
              >
                {t(`customer.tab_${key}`)}
              </button>
            ))}
          </div>

          {error ? (
            <Notice variant="error">{errorMessage(error, t('common.somethingWrong'))}</Notice>
          ) : null}

          {isLoading ? <p className={INK_62}>{t('common.loading')}</p> : null}

          {!isLoading && shown.length === 0 && !error ? (
            <EmptyState>
              <p>{t('customer.noTickets')}</p>
              <LinkButton to={PATHS.home} className="mt-3 inline-flex">
                {t('customer.browseFilms')}
              </LinkButton>
            </EmptyState>
          ) : null}

          <ul className="m-0 list-none p-0">
            {shown.map((order) => (
              <li key={order.id} className="mb-2.5">
                <button
                  type="button"
                  onClick={() => setOpenedId(order.id)}
                  className={`flex w-full cursor-pointer items-center gap-3 rounded-xl border bg-transparent p-3.5 text-left transition-colors duration-fast ease-out hover-fine:border-brand ${INK_BORDER_14} ${INK_BG_035}`}
                >
                  <span
                    className={`h-2.5 w-2.5 flex-none rounded-full ${ROW_BADGE_CLASS[order.status]}`}
                    aria-hidden="true"
                  />
                  <span className="min-w-0 flex-1">
                    <span className={`block truncate text-[15px] font-bold ${INK}`}>
                      {order.showtime?.movie_title ?? '—'}
                    </span>
                    <span className={`block truncate text-[13px] tabular-nums ${INK_60}`}>
                      {formatDateTime(order.showtime?.start_at)}
                      {order.showtime?.hall_name ? ` · ${order.showtime.hall_name}` : null}
                    </span>
                  </span>
                  <span className="flex-none text-right">
                    <span className={`block text-sm font-bold tabular-nums ${INK}`}>
                      {formatVND(order.payable_amount)}
                    </span>
                    <span className={`block text-xs ${INK_60}`}>
                      {t(`booking.status_${order.status}`)}
                    </span>
                  </span>
                  <span className={`flex-none text-lg ${INK_60}`} aria-hidden="true">
                    ›
                  </span>
                </button>
              </li>
            ))}
          </ul>

          {tab === 'history' && !showAllHistory && hasOlderHistory ? (
            <div className="mt-6 flex justify-center">
              <Button variant="ghost" onClick={() => setShowAllHistory(true)}>
                {t('customer.loadMoreHistory')}
              </Button>
            </div>
          ) : null}
        </>
      )}
    </section>
  );
};

/** Single-order detail: full tickets (QR) + quick rebook. */
const TicketDetail = ({
  order,
  isHistory,
  onBack,
}: {
  order: OrderStatus;
  isHistory: boolean;
  onBack: () => void;
}) => {
  const { t } = useTranslation();
  // The outer list omits tickets, so the detail fetches them.
  const detail = useOrderDetail(order.id);
  const tickets = detail.data?.tickets ?? [];

  return (
    <div>
      <Button variant="ghost" onClick={onBack} className="mb-4">
        ← {t('customer.back')}
      </Button>

      {detail.error ? (
        <Notice variant="error">{errorMessage(detail.error, t('common.somethingWrong'))}</Notice>
      ) : null}
      {detail.isLoading ? <p className={INK_62}>{t('common.loading')}</p> : null}

      {detail.data ? (
        <div className="flex max-w-140 flex-col gap-2.5">
          <TicketCard order={detail.data} tickets={tickets} showRebook={isHistory} />
        </div>
      ) : null}
    </div>
  );
};

export default TicketsSection;
