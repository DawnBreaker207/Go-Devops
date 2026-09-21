import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';
import { useHasRole } from '@/hooks/useHasRole';
import { useMyOrders } from '@/features/booking-flow/hooks/useOrders';
import { accountTicketsPath } from '@/routes/paths';
import { toCinemaTime } from '@/utils/format';
import { INK_65, INK_BG_04, INK_BORDER_16 } from '@/theme/customerTw';

/** Shows signed-in customers their next unstarted `confirmed` order, else hides. Reuses useMyOrders; `showtime.started` is backend-computed, never derived client-side. */
export const UpcomingTicketTeaser = () => {
  const { t } = useTranslation();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isCustomer = useHasRole('customer');
  const enabled = isAuthenticated && isCustomer;

  const { data } = useMyOrders({ page: 1, page_size: 20 }, enabled);

  const nextTicket = useMemo(() => {
    if (!enabled) return null;
    const upcoming = (data?.items ?? [])
      .filter((order) => order.status === 'confirmed' && order.showtime && !order.showtime.started)
      .sort((a, b) => (a.showtime!.start_at < b.showtime!.start_at ? -1 : 1));
    return upcoming[0] ?? null;
  }, [enabled, data]);

  if (!enabled || !nextTicket || !nextTicket.showtime) return null;

  const { showtime } = nextTicket;

  return (
    <Link
      to={accountTicketsPath()}
      className={[
        'mb-5 flex items-center justify-between gap-3 rounded-xl border px-4 py-3.5 text-inherit no-underline',
        INK_BORDER_16,
        INK_BG_04,
        'transition-[border-color,transform] duration-fast ease-out active:scale-(--motion-scale-press) hover-fine:border-brand',
      ].join(' ')}
    >
      <div>
        <span className="mb-0.5 block text-[11px] font-bold tracking-wide text-brand uppercase">
          {t('customer.upcomingTicketTitle')}
        </span>
        <div className="text-[15px] font-bold">{showtime.movie_title}</div>
        <div className={`text-xs ${INK_65}`}>
          {toCinemaTime(showtime.start_at).format('DD/MM HH:mm')} · {showtime.hall_name}
        </div>
      </div>
      <span
        className="flex-none text-[13px] font-semibold whitespace-nowrap text-brand"
        aria-hidden="true"
      >
        {t('customer.upcomingTicketCta')} →
      </span>
    </Link>
  );
};

export default UpcomingTicketTeaser;
