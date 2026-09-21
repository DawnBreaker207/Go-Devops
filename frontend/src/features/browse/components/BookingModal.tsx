import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { useMovieShowtimesRange } from '../hooks/useBrowse';
import { seatMapApi } from '@/api/seatmap.api';
import { selectSeatPath } from '@/routes/paths';
import { useAuthCheckpoint } from '@/features/auth/useAuthCheckpoint';
import LoginBottomSheet from '@/features/auth/LoginBottomSheet';
import Notice from '@/components/ui/Notice';
import Button from '@/components/ui/Button';
import { API_DATE_FORMAT, CINEMA_TZ, toCinemaTime } from '@/utils/format';
import { errorMessage } from '@/utils/error';
import { INK_60 } from '@/theme/customerTw';
import type { ShowtimeListItem } from '@/types';

export interface BookingModalProps {
  movieId: string;
  movieTitle: string;
  onClose: () => void;
}

const DAYS_AHEAD = 7;

const CalendarIcon = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <rect x="3.5" y="5" width="17" height="15" rx="2" stroke="currentColor" strokeWidth="1.6" />
    <path
      d="M3.5 9.5h17M8 3v3.5M16 3v3.5"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
    />
  </svg>
);

const CheckIcon = () => (
  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path
      d="M5 13l5 5L19 7"
      stroke="currentColor"
      strokeWidth="3"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

/** Showtime picker modal from "Book" on a movie card. One scrollable day-grouped list (3-col time grid); picking highlights only, "Confirm" below advances. Unlike FilmPage (day tabs), this has no tabs; both UIs stay. Backend returns ONE day per call, so fan out one query/day over DAYS_AHEAD and merge client-side (useMovieShowtimesRange). No per-show seat counts from the API, so no "x/y seats" line or sold-out disabled state; true sellout is detected on Confirm via seatMapApi.forShowtime. */
export const BookingModal = ({ movieId, movieTitle, onClose }: BookingModalProps) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { requireAuth, sheetOpen, contextMessage, closeSheet, handleSheetSuccess } =
    useAuthCheckpoint();

  const [shown, setShown] = useState(false);
  const [selected, setSelected] = useState<ShowtimeListItem | null>(null);
  const [seatCheckError, setSeatCheckError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);

  const dates = useMemo(
    () =>
      Array.from({ length: DAYS_AHEAD }, (_, i) =>
        dayjs().tz(CINEMA_TZ).startOf('day').add(i, 'day').format(API_DATE_FORMAT)
      ),
    []
  );
  const { groups, isFetching, error } = useMovieShowtimesRange(movieId, dates);

  useEffect(() => {
    const raf = requestAnimationFrame(() => setShown(true));
    return () => cancelAnimationFrame(raf);
  }, []);

  const close = () => {
    setShown(false);
    onClose();
  };

  const dateLabel = (date: string) => {
    const today = dates[0];
    const tomorrow = dates[1];
    if (date === today) return t('customer.today');
    if (date === tomorrow) return t('customer.tomorrow');
    return dayjs(date, API_DATE_FORMAT).format('DD/MM');
  };

  const confirmSelection = () => {
    if (!selected) return;
    setSeatCheckError(null);
    const message = t('customer.loginToSelectSeat', {
      time: toCinemaTime(selected.start_at).format('HH:mm'),
      hall: selected.hall_name,
    });

    requireAuth(() => {
      setConfirming(true);
      void seatMapApi
        .forShowtime(selected.id)
        .then((seatMap) => {
          const stillHasSeat = seatMap.seats.some(
            (seat) => !seat.is_gap && seat.showtime_seat_id && seat.status === 'available'
          );
          if (!stillHasSeat) {
            setSeatCheckError(t('customer.showtimeSoldOut'));
            setConfirming(false);
            return;
          }
          navigate(selectSeatPath(selected.id));
        })
        .catch(() => navigate(selectSeatPath(selected.id)));
    }, message);
  };

  return (
    <>
      <div
        role="presentation"
        onClick={close}
        className={`fixed inset-0 z-100 flex items-end justify-center bg-black/60 transition-opacity duration-moderate ease-out sm:items-center ${
          shown ? 'opacity-100' : 'opacity-0'
        }`}
      >
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="cp-booking-modal-title"
          onClick={(event) => event.stopPropagation()}
          className={`relative flex max-h-[85vh] w-full max-w-135 flex-col overflow-hidden rounded-t-2xl bg-(--cp-backdrop-mid) px-5 pt-4 pb-6 text-white shadow-2xl transition-transform duration-moderate ease-out sm:mb-0 sm:rounded-2xl ${
            shown ? 'translate-y-0' : 'translate-y-full sm:translate-y-4'
          }`}
        >
          <div className="mx-auto mb-2 h-1 w-10 flex-none rounded-full bg-white/20 sm:hidden" />

          <div className="mb-4 flex flex-none items-start justify-between gap-3">
            <h2 id="cp-booking-modal-title" className="text-lg font-bold text-brand">
              {movieTitle}
            </h2>
            <button
              type="button"
              aria-label={t('common.cancel')}
              onClick={close}
              className="shrink-0 cursor-pointer border-none bg-transparent p-1 text-lg leading-none text-white/60"
            >
              ✕
            </button>
          </div>

          {seatCheckError ? (
            <Notice variant="error" className="mb-3 flex-none">
              {seatCheckError}
            </Notice>
          ) : null}

          {error ? (
            <Notice variant="error" className="flex-none">
              {errorMessage(error, t('common.somethingWrong'))}
            </Notice>
          ) : null}

          <div className="-mr-2 flex-1 overflow-y-auto pr-2">
            {isFetching && groups.length === 0 ? (
              <p className={`text-sm ${INK_60}`}>{t('common.loading')}</p>
            ) : null}

            {!isFetching && groups.length === 0 && !error ? (
              <div className={`flex flex-col items-center gap-2 py-10 text-center ${INK_60}`}>
                <CalendarIcon />
                <p>{t('customer.noShowtimesRange')}</p>
              </div>
            ) : null}

            {groups.map((group) => (
              <div key={group.date} className="mb-6">
                <div className="mb-3 flex items-center gap-2">
                  <span className="rounded bg-brand px-2 py-1 text-xs font-bold text-on-brand">
                    {t('customer.dateLabel')}
                  </span>
                  <span className="text-sm font-bold text-white/85">{dateLabel(group.date)}</span>
                </div>

                <div className="grid grid-cols-3 gap-3">
                  {group.items.map((show) => {
                    const isSelected = selected?.id === show.id;
                    return (
                      <button
                        key={show.id}
                        type="button"
                        onClick={() => setSelected(show)}
                        aria-pressed={isSelected}
                        className="relative flex cursor-pointer flex-col items-center gap-0.5 rounded-lg border-2 border-brand/45 bg-brand/10 p-3 text-white transition-colors duration-fast ease-out hover-fine:border-brand hover-fine:bg-brand/20"
                      >
                        <span className="text-sm font-semibold">
                          {toCinemaTime(show.start_at).format('HH:mm')}
                        </span>
                        <span className="text-[11px] text-white/55">{show.hall_name}</span>
                        {isSelected ? (
                          <span className="absolute -top-2 -right-2 flex h-5 w-5 items-center justify-center rounded-full bg-brand text-on-brand">
                            <CheckIcon />
                          </span>
                        ) : null}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>

          <Button
            variant="primary"
            block
            className="mt-4 h-11 flex-none text-base font-bold"
            disabled={!selected || confirming}
            onClick={confirmSelection}
          >
            {t('common.confirm')}
          </Button>
        </div>
      </div>

      {sheetOpen ? (
        <LoginBottomSheet
          contextMessage={contextMessage}
          onClose={closeSheet}
          onSuccess={handleSheetSuccess}
        />
      ) : null}
    </>
  );
};

export default BookingModal;
