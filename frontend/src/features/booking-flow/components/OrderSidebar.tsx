import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useOrderDetail } from '../hooks/useOrders';
import { useMovieDetail } from '@/features/browse/hooks/useBrowse';
import { useBookingFlowStore } from '@/stores/bookingFlowStore';
import { useCountdown } from '../useCountdown';
import { errorMessage } from '@/utils/error';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { INK_60, INK_BG_035, INK_BORDER_14 } from '@/theme/customerTw';
import BookingSummarySidebar from './BookingSummarySidebar';
import type { BookingStep } from './BookingStepper';

interface OrderSidebarProps {
  bookingId: string | undefined;
  step: BookingStep;
  /** Seats already known from the previous step (passed down) - beats navigation state. */
  seatLabels?: string[];
  onBack?: () => void;
  onContinue?: () => void;
  continueDisabled?: boolean;
  continueLoading?: boolean;
  continueLabel?: string;
}

/** Summary sidebar on live order data: movie/hall/time/total/countdown (pending only). Seat labels: prop -> store -> navigation state -> issued tickets -> placeholder. */
export const OrderSidebar = ({
  bookingId,
  step,
  seatLabels: seatLabelsProp,
  onBack,
  onContinue,
  continueDisabled,
  continueLoading,
  continueLabel,
}: OrderSidebarProps) => {
  const location = useLocation();
  const { t } = useTranslation();
  const order = useOrderDetail(bookingId);
  const movie = useMovieDetail(order.data?.showtime?.movie_id);
  // Shared store first (survives navigation + F5), then navigation state (lost on F5), then issued tickets.
  const storedSeats = useBookingFlowStore((s) => s.seats);
  // Countdown only for pending orders WITH a hold - pass undefined and the sidebar hides it instead of showing fake 00:00.
  const secondsLeft = useCountdown(
    order.data?.status === 'pending' ? order.data.expires_at : undefined
  );

  // The sidebar never disappears: no order -> waiting frame + Back; error -> message + retry + Back; full content only on data.
  if (!bookingId) {
    return (
      <div className="flex flex-col gap-2.5">
        <Notice variant="info" role="status">
          {t('customer.orderNotReady')}
        </Notice>
        {onBack ? (
          <Button variant="ghost" block onClick={onBack}>
            ← {t('customer.back')}
          </Button>
        ) : null}
      </div>
    );
  }
  if (order.isLoading) {
    return (
      <aside
        className={`w-full rounded-2xl border p-4 lg:sticky lg:top-24 lg:w-95 ${INK_BORDER_14} ${INK_BG_035}`}
      >
        <p className={`m-0 text-sm ${INK_60}`}>{t('common.loading')}</p>
      </aside>
    );
  }
  if (order.error) {
    return (
      <div className="flex flex-col gap-2.5">
        <Notice variant="error">{errorMessage(order.error, t('common.somethingWrong'))}</Notice>
        <Button
          variant="ghost"
          block
          onClick={() => void order.refetch()}
          disabled={order.isFetching}
        >
          {t('common.retry')}
        </Button>
        {onBack ? (
          <Button variant="ghost" block onClick={onBack}>
            ← {t('customer.back')}
          </Button>
        ) : null}
      </div>
    );
  }
  if (!order.data) {
    return (
      <div className="flex flex-col gap-2.5">
        <Notice variant="info" role="status">
          {t('customer.orderNotReady')}
        </Notice>
        {onBack ? (
          <Button variant="ghost" block onClick={onBack}>
            ← {t('customer.back')}
          </Button>
        ) : null}
      </div>
    );
  }

  const o = order.data;
  const stateLabels = (location.state as { seatLabels?: string[] } | null)?.seatLabels;
  const seatLabels =
    seatLabelsProp ??
    (storedSeats.length > 0 ? storedSeats.map((s) => s.label) : undefined) ??
    stateLabels ??
    o.tickets.map((ticket) => ticket.seat_label);

  return (
    <BookingSummarySidebar
      posterUrl={movie.data?.poster_url}
      movieTitle={o.showtime?.movie_title ?? ''}
      genre={movie.data?.genre}
      hallName={o.showtime?.hall_name ?? ''}
      startAt={o.showtime?.start_at ?? ''}
      seatLabels={seatLabels}
      total={o.total_amount}
      step={step}
      onBack={onBack}
      onContinue={onContinue}
      continueDisabled={continueDisabled}
      continueLoading={continueLoading}
      continueLabel={continueLabel}
      countdownSeconds={o.status === 'pending' && o.expires_at ? secondsLeft : undefined}
    />
  );
};

export default OrderSidebar;
