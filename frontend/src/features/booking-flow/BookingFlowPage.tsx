import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { seatMapApi, SEATMAP_QUERY_KEY } from '@/api/seatmap.api';
import { orderApi } from '@/api/order.api';
import { buildGridLayout, groupSeatsByRow, seatsPerRowFromSeats } from '@/features/hall/seatGrid';
import {
  ORDER_QUERY_KEY,
  useHoldSeats,
  useInitOrder,
  useOrderDetail,
  useRefreshOrder,
} from './hooks/useOrders';
import { useCombos, useCreateComboOrder } from './hooks/useCombos';
import { useSeatMapRealtime, type SeatStatusUpdate } from './hooks/useSeatMapRealtime';
import { useMovieDetail } from '@/features/browse/hooks/useBrowse';
import { useCountdown } from './useCountdown';
import { useHasRole } from '@/hooks/useHasRole';
import { useAuthStore } from '@/stores/authStore';
import { MAX_FLOW_SEATS, useBookingFlowStore, type FlowSeat } from '@/stores/bookingFlowStore';
import { PATHS, bookingSuccessPath } from '@/routes/paths';
import type { SeatMap, SeatMapSeat } from '@/types';
import type { ComboOrderItemPayload } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { INK_62, INK_65 } from '@/theme/customerTw';
import BookingLayout from './components/BookingLayout';
import OrderSidebar from './components/OrderSidebar';
import BookingSummarySidebar from './components/BookingSummarySidebar';
import type { StepDef } from './components/BookingStepper';
import SeatStep from './steps/SeatStep';
import ComboStep from './steps/ComboStep';
import CheckoutStep from './steps/CheckoutStep';

/** Max combos per order (backend `max=20`). */
const MAX_QTY = 20;

type FlowStep = 0 | 1 | 2;

/** Single-route booking flow (seats→combo→checkout); confirmed orders go to the success page; parent holds picks, holds, combos and the recovery order. */
export const BookingFlowPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showtimeId } = useParams<{ showtimeId: string }>();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const queryClient = useQueryClient();
  // Holds are RequireRoles(customer): operators see the map but pay 403s. Say so upfront instead of on click.
  const isCustomer = useHasRole('customer');

  const storeShowtimeId = useBookingFlowStore((s) => s.showtimeId);
  const storeBookingId = useBookingFlowStore((s) => s.bookingId);
  const storeSeats = useBookingFlowStore((s) => s.seats);
  const enterShowtime = useBookingFlowStore((s) => s.enterShowtime);
  const setBooking = useBookingFlowStore((s) => s.setBooking);

  const [step, setStep] = useState<FlowStep | null>(null);
  // Counts sidebar "Pay" presses at checkout - CheckoutStep listens and pays (not a ref). Monotonic, so one press never fires twice.
  const [paySignal, setPaySignal] = useState(0);
  // Navigate by handler only (no setState in effects): current step = user
  // pick, else derived from the saved order.
  const goStep = useCallback((n: FlowStep) => setStep(n), []);

  // Entering with a different showtime = new run: ignore the old booking (store writes happen on successful hold, in the handler).
  const bookingId =
    showtimeId && showtimeId !== storeShowtimeId ? undefined : (storeBookingId ?? undefined);

  // Recovery order (F5 / opened from My tickets): confirmed -> SEPARATE
  // SUCCESS page; pending WITH seats -> checkout; empty pending (fresh entry
  // shell) -> stay on seats. Manual `step` wins once set.
  const savedOrder = useOrderDetail(bookingId);
  // A just-created entry order (seatless) must not auto-jump to checkout -
  // state (not ref) so render can read it.
  const [entryFresh, setEntryFresh] = useState(false);
  const autoStep: FlowStep =
    !bookingId || savedOrder.error || !savedOrder.data || savedOrder.data.status === 'confirmed'
      ? 0
      : savedOrder.data.status === 'pending' && !entryFresh && storeSeats.length > 0
        ? 2
        : 0;
  const effectiveStep: FlowStep = step ?? autoStep;

  // Warn on tab close mid-flow (picking seats or holding an order).
  const pendingOrder = savedOrder.data?.status === 'pending' ? savedOrder.data : null;

  // --- Step 0: map + pick; ORDER OPENS AT ENTRY (init), hold on Continue ---
  // Entry opens an empty order + countdown; clicks only flip local state; hold attaches at step change. Deselect-all cancels the order, killing the countdown.
  const selectedIds = useBookingFlowStore((s) => s.selectedIds);
  const toggleSeat = useBookingFlowStore((s) => s.toggleSeat);
  const clearBooking = useBookingFlowStore((s) => s.clearBooking);
  const clearAll = useBookingFlowStore((s) => s.clear);
  const [holdError, setHoldError] = useState<string | null>(null);
  const [flowError, setFlowError] = useState<string | null>(null);
  const hold = useHoldSeats();
  const initOrder = useInitOrder();
  const refreshOrder = useRefreshOrder();
  const initTriedRef = useRef<string | null>(null);

  const canWatchSeats = Boolean(showtimeId) && isAuthenticated;

  const applySeatUpdates = useCallback(
    (updates: SeatStatusUpdate[]) => {
      if (!showtimeId || updates.length === 0) return;
      const byId = new Map(updates.map((u) => [u.id, u.status]));
      queryClient.setQueryData<SeatMap>([SEATMAP_QUERY_KEY, showtimeId], (old) => {
        if (!old) return old;
        return {
          ...old,
          seats: old.seats.map((s) =>
            s.showtime_seat_id && byId.has(s.showtime_seat_id)
              ? { ...s, status: byId.get(s.showtime_seat_id) as SeatMapSeat['status'] }
              : s
          ),
        };
      });
    },
    [queryClient, showtimeId]
  );

  // Primary seat-state source (hook owns the SSE protocol); `sseDown` flags dead realtime.
  const { sseDown } = useSeatMapRealtime(showtimeId, canWatchSeats, applySeatUpdates);

  const seatMap = useQuery({
    queryKey: [SEATMAP_QUERY_KEY, showtimeId],
    queryFn: () => seatMapApi.forShowtime(showtimeId as string),
    enabled: canWatchSeats,
    // SSE is the live source; long staleTime since this is only fallback for remounts / first entry.
    staleTime: 60_000,
    // sseDown: SSE broken (weak net, mid-flow token expiry...) - re-poll so the map never freezes waiting on SSE recovery.
    refetchInterval: sseDown ? 5000 : false,
  });

  const seats = useMemo(() => seatMap.data?.seats ?? [], [seatMap.data]);
  const rows = useMemo(() => groupSeatsByRow(seats), [seats]);
  const layout = useMemo(
    () => buildGridLayout(seatsPerRowFromSeats(seats), seatMap.data?.aisle_after_cols ?? []),
    [seats, seatMap.data?.aisle_after_cols]
  );

  const selectedSeats = useMemo(
    () => seats.filter((s) => s.showtime_seat_id && selectedIds.includes(s.showtime_seat_id)),
    [seats, selectedIds]
  );
  const selected = useMemo(() => new Set(selectedIds), [selectedIds]);
  const total = selectedSeats.reduce((sum, s) => sum + s.price, 0);

  // Poster/genre for the sidebar header - SeatMap omits both, fetched separately by movie_id.
  const movieDetail = useMovieDetail(seatMap.data?.movie_id);

  // Step-0 countdown reads the held (pending) order - hidden pre-hold.
  const step0Countdown = useCountdown(
    bookingId && savedOrder.data?.status === 'pending' ? savedOrder.data.expires_at : undefined
  );

  const doHold = useCallback(
    async (details: FlowSeat[]): Promise<boolean> => {
      if (!showtimeId || details.length === 0) return false;
      setHoldError(null);
      try {
        const result = await hold.mutateAsync({
          show_id: showtimeId,
          // MUST be showtime_seat_id. seats.id 400s ("seat does not belong to this showtime").
          seat_ids: details.map((d) => d.seatId),
        });
        // enterShowtime BEFORE setBooking (resets only on showtime change, never wipes same-showtime temp picks).
        enterShowtime(showtimeId);
        setBooking(result.booking_id, details);
        // Hold mints a new order (uncached query): prefetch while the button loads so combo step has data (else page + sidebar flash). Prefetch errors are fine (sidebar retries).
        await queryClient.prefetchQuery({
          queryKey: [ORDER_QUERY_KEY, 'detail', result.booking_id],
          queryFn: () => orderApi.detail(result.booking_id),
        });
        return true;
      } catch (error) {
        // Mid-flow 409: someone else grabbed a seat. Keep picks so the customer swaps seats.
        setHoldError(errorMessage(error, t('common.somethingWrong')));
        void seatMap.refetch();
        return false;
      }
    },
    [showtimeId, hold, setBooking, enterShowtime, queryClient, seatMap, t]
  );

  // Entry: open an empty order so countdown runs immediately (idempotent: a live order returns as-is). StrictMode double-fires the effect in dev - ref-guarded.
  useEffect(() => {
    if (!showtimeId || !isCustomer || bookingId) return;
    if (initTriedRef.current === showtimeId) return;
    initTriedRef.current = showtimeId;
    initOrder
      .mutateAsync(showtimeId)
      .then((res) => {
        // Fresh entry orders (seatless) must not auto-jump to checkout.
        if (!res.reused) setEntryFresh(true);
        enterShowtime(showtimeId);
        setBooking(res.booking_id, []);
      })
      .catch((error) => {
        setHoldError(errorMessage(error, t('common.somethingWrong')));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showtimeId, isCustomer, bookingId]);

  // Continue: ALWAYS hold picked seats first (empty init or old order both get
  // replaced), then combo. Never skip the hold.
  const handleProceed = useCallback(async () => {
    const ok = await doHold(
      selectedIds.map((id) => {
        const s = seats.find((seat) => seat.showtime_seat_id === id);
        return {
          seatId: id,
          label: s?.label ?? id,
          seatType: s?.seat_type ?? 'standard',
          price: s?.price ?? 0,
        };
      })
    );
    if (ok) goStep(1);
  }, [doHold, selectedIds, seats, goStep]);

  // Silent 45s heartbeat: extends within the lifetime cap. Failures (cap hit,
  // seats lost, showtime closed) -> back to step 0 with an error, beat stops.
  const refreshRef = useRef(() => {});
  useEffect(() => {
    refreshRef.current = () => {
      if (!bookingId) return;
      refreshOrder.mutateAsync(bookingId).catch((error) => {
        clearBooking();
        setFlowError(errorMessage(error, t('common.somethingWrong')));
        goStep(0);
      });
    };
  });
  useEffect(() => {
    if (!bookingId) return;
    const id = window.setInterval(() => refreshRef.current(), 45_000);
    return () => window.clearInterval(id);
  }, [bookingId, effectiveStep]);

  // Expiry on steps 0/1: home (sweeper reaps the order). Step 2 never
  // self-navigates (gateway tab may be open elsewhere) - the screen shows expiry itself.
  const expiredClient =
    bookingId != null &&
    savedOrder.data?.status === 'pending' &&
    step0Countdown === 0 &&
    savedOrder.data?.expires_at !== undefined;
  useEffect(() => {
    if (expiredClient && (effectiveStep === 0 || effectiveStep === 1)) {
      clearAll();
      navigate(PATHS.home, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expiredClient]);

  // --- Step 1: combo ---
  const combos = useCombos();
  const createOrder = useCreateComboOrder();
  const [qty, setQty] = useState<Record<string, number>>({});
  const [comboError, setComboError] = useState<string | null>(null);

  const items: ComboOrderItemPayload[] = useMemo(
    () =>
      Object.entries(qty)
        .filter(([, quantity]) => quantity > 0)
        .map(([combo_id, quantity]) => ({ combo_id, quantity })),
    [qty]
  );

  const changeQty = useCallback((comboId: string, delta: number) => {
    setQty((current) => {
      const next = Math.min(MAX_QTY, Math.max(0, (current[comboId] ?? 0) + delta));
      return { ...current, [comboId]: next };
    });
  }, []);

  const handleComboContinue = useCallback(async () => {
    if (!bookingId || items.length === 0) {
      goStep(2);
      return;
    }
    setComboError(null);
    try {
      await createOrder.mutateAsync({ booking_id: bookingId, items });
      goStep(2);
    } catch (error) {
      // Combo-order errors must never block the ticket order - flag it so a
      // retry keeps items (pressing Continue again re-tries).
      setComboError(errorMessage(error, t('customer.comboOrderError')));
    }
  }, [bookingId, items, createOrder, goStep, t]);

  // --- Step navigation ---
  // Confirming belongs to /payment-result post-gateway - no outbound events from CheckoutStep anymore.

  useEffect(() => {
    const guard = (event: BeforeUnloadEvent) => {
      if (selected.size > 0 || pendingOrder) event.preventDefault();
    };
    window.addEventListener('beforeunload', guard);
    return () => window.removeEventListener('beforeunload', guard);
  }, [selected.size, pendingOrder]);

  if (!showtimeId) {
    return <Navigate to={PATHS.home} replace />;
  }

  if (!isAuthenticated) {
    // /shows/:id/seats needs JWT, so unauthenticated users can't fetch it.
    return (
      <Notice variant="info" role="status">
        <span>{t('customer.seatsNeedLogin')}</span>
      </Notice>
    );
  }

  // Confirmed orders (reopened old order / F5 at confirm time / stale store run)
  // never stay in-flow - straight to the separate success page.
  if (bookingId && !savedOrder.isLoading && savedOrder.data?.status === 'confirmed') {
    return <Navigate to={bookingSuccessPath(bookingId)} replace />;
  }

  const steps: StepDef[] = [
    { n: 1, label: t('customer.stepSeats'), icon: '▦' },
    { n: 2, label: t('customer.stepCombo'), icon: '🍿' },
    { n: 3, label: t('customer.stepConfirm'), icon: '👤' },
  ];

  if (seatMap.error && effectiveStep === 0) {
    // 404 here means two things ("missing" and "no longer on sale") under one
    // 40400 code, distinguishable only by the backend's English sentence - so
    // render it verbatim instead of guessing.
    return (
      <Notice variant="error">{errorMessage(seatMap.error, t('common.somethingWrong'))}</Notice>
    );
  }
  // Wait for the seat map (step 0) or the recovery order before painting anything.
  if ((seatMap.isLoading && effectiveStep === 0) || (bookingId && savedOrder.isLoading)) {
    return <p className={INK_62}>{t('common.loading')}</p>;
  }
  if (effectiveStep === 0 && !seatMap.data) return null;

  const show = seatMap.data;

  const sidebarStep0 = show ? (
    <BookingSummarySidebar
      posterUrl={movieDetail.data?.poster_url}
      movieTitle={show.movie_title}
      genre={movieDetail.data?.genre}
      hallName={show.hall_name}
      startAt={show.start_at}
      seatLabels={selectedSeats.map((s) => s.label)}
      total={total}
      step={1}
      onContinue={() => void handleProceed()}
      continueDisabled={selectedSeats.length === 0 || !isCustomer || hold.isPending}
      continueLoading={hold.isPending}
      countdownSeconds={bookingId ? step0Countdown : undefined}
    />
  ) : (
    <></>
  );

  return (
    <BookingLayout
      current={(effectiveStep + 1) as 1 | 2 | 3}
      steps={steps}
      onStepBack={(n) => goStep((n - 1) as FlowStep)}
      sidebar={
        effectiveStep === 0 ? (
          sidebarStep0
        ) : (
          <OrderSidebar
            bookingId={bookingId}
            step={(effectiveStep + 1) as 1 | 2 | 3}
            onBack={
              effectiveStep === 1
                ? () => goStep(0)
                : effectiveStep === 2
                  ? () => goStep(1)
                  : undefined
            }
            onContinue={
              effectiveStep === 1
                ? () => void handleComboContinue()
                : effectiveStep === 2
                  ? () => setPaySignal((n) => n + 1)
                  : undefined
            }
            continueDisabled={effectiveStep === 1 && createOrder.isPending}
            continueLoading={effectiveStep === 1 && createOrder.isPending}
          />
        )
      }
      mobileBar={
        effectiveStep === 0 ? (
          <>
            <div className="min-w-0 flex-1">
              <span className={`block text-[11px] tracking-[1px] ${INK_65}`}>
                {t('customer.seatCount', { count: selectedSeats.length, max: MAX_FLOW_SEATS })}
              </span>
              <span className="block truncate text-sm font-bold tabular-nums">
                {formatVND(total)}
              </span>
            </div>
            <Button
              disabled={selectedSeats.length === 0 || !isCustomer || hold.isPending}
              onClick={() => void handleProceed()}
            >
              {hold.isPending ? t('common.loading') : t('customer.proceed')}
            </Button>
          </>
        ) : undefined
      }
      main={
        effectiveStep === 0 && show ? (
          <SeatStep
            show={show}
            rows={rows}
            layout={layout}
            selected={selected}
            holdError={flowError ?? holdError}
            isCustomer={isCustomer}
            onToggle={toggleSeat}
          />
        ) : effectiveStep === 1 ? (
          <ComboStep
            combos={combos.data ?? []}
            combosLoading={combos.isLoading}
            qty={qty}
            maxQty={MAX_QTY}
            comboError={comboError}
            onQty={changeQty}
          />
        ) : effectiveStep === 2 && bookingId ? (
          <CheckoutStep bookingId={bookingId} paySignal={paySignal} />
        ) : null
      }
    />
  );
};

export default BookingFlowPage;
