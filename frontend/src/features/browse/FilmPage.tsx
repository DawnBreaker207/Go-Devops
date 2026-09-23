import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { useMovieDetail, useMovieShowtimes } from './hooks/useBrowse';
import { selectSeatPath } from '@/routes/paths';
import { seatMapApi } from '@/api/seatmap.api';
import { errorMessage } from '@/utils/error';
import {
  API_DATE_FORMAT,
  CINEMA_TZ,
  formatDuration,
  formatVND,
  toCinemaTime,
} from '@/utils/format';
import { useAuthCheckpoint } from '@/features/auth/useAuthCheckpoint';
import LoginBottomSheet from '@/features/auth/LoginBottomSheet';
import ComingSoonHero from './components/ComingSoonHero';
import {
  showtimeCardClass,
  showtimeHallClass,
  showtimePriceClass,
  showtimeTimeClass,
} from './components/showtimeStyles';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import {
  INK,
  INK_55,
  INK_62,
  INK_80,
  INK_85,
  INK_BG_06,
  INK_BORDER_22,
  INK_BORDER_25,
} from '@/theme/customerTw';
import type { ShowtimeListItem } from '@/types';

/** Movie detail + showtime picker. GET /movies/:id/showtimes is clamped to ONE day (today by default), so the day strip is mandatory. */

/** Days offered on the day strip. */
const DAYS_AHEAD = 7;

/** Age-badge colors; see --cp-rating-* in index.css. */
const ratingVars = (rating: string) => ({
  background: `var(--cp-rating-${rating.toLowerCase()})`,
  color: `var(--cp-rating-${rating.toLowerCase()}-text)`,
});

export const FilmPage = () => {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { requireAuth, sheetOpen, contextMessage, closeSheet, handleSheetSuccess } =
    useAuthCheckpoint();

  // Work days in CINEMA time, not viewer time: dayjs() elsewhere shifts "today" for out-of-zone viewers.
  const days = useMemo(
    () =>
      Array.from({ length: DAYS_AHEAD }, (_, i) =>
        dayjs().tz(CINEMA_TZ).startOf('day').add(i, 'day')
      ),
    []
  );
  const [selectedDay, setSelectedDay] = useState(() => days[0].format(API_DATE_FORMAT));
  const [seatCheckError, setSeatCheckError] = useState<string | null>(null);

  const movie = useMovieDetail(id);
  const showtimes = useMovieShowtimes(id, selectedDay);

  // Single login-checkpoint entry: with token requireAuth navigates straight; without, it opens LoginBottomSheet for that show. Closing mid-sheet keeps the selection (still on FilmPage, no route push). Seat-map needs JWT, so availability is rechecked only AFTER auth (a show can sell out mid-password-entry).
  const handleSelectSeat = (show: ShowtimeListItem) => {
    setSeatCheckError(null);
    const message = t('customer.loginToSelectSeat', {
      time: toCinemaTime(show.start_at).format('HH:mm'),
      hall: show.hall_name,
    });

    requireAuth(() => {
      void seatMapApi
        .forShowtime(show.id)
        .then((seatMap) => {
          const stillHasSeat = seatMap.seats.some(
            (seat) => !seat.is_gap && seat.showtime_seat_id && seat.status === 'available'
          );
          if (!stillHasSeat) {
            setSeatCheckError(t('customer.showtimeSoldOut'));
            return;
          }
          navigate(selectSeatPath(show.id));
        })
        .catch(() => {
          // Seat state unreadable (network, just-closed show): still enter SelectSeatPage; it handles its own errors.
          navigate(selectSeatPath(show.id));
        });
    }, message);
  };

  if (movie.error) {
    return <Notice variant="error">{errorMessage(movie.error, t('common.somethingWrong'))}</Notice>;
  }

  const film = movie.data;

  return (
    <div className="relative">
      {film?.poster_url ? (
        <div
          className="absolute -top-2 -right-6 -left-6 z-0 h-105 mask-[linear-gradient(to_bottom,black,transparent)] scale-[1.15] bg-cover bg-position-[center_20%] opacity-35 blur-2xl saturate-[1.1] [-webkit-mask-image:linear-gradient(to_bottom,black,transparent)] pointer-events-none"
          style={{ backgroundImage: `url(${film.poster_url})` }}
          aria-hidden="true"
        />
      ) : null}

      <div className="relative z-10 mt-4 grid grid-cols-[280px_minmax(0,1fr)] gap-8 max-[760px]:grid-cols-1 max-[760px]:gap-5">
        <div
          className={`relative aspect-2/3 overflow-hidden rounded-xl ${INK_BG_06} shadow-[0_24px_48px_rgba(0,0,0,0.5),0_0_0_1px_rgba(var(--cp-ink-rgb),0.08)] max-[760px]:max-w-55`}
        >
          {film?.poster_url ? (
            <img
              src={film.poster_url}
              alt=""
              loading="eager"
              className="block h-full w-full object-cover"
            />
          ) : null}
        </div>

        <div>
          <h1 className="mb-2 text-[30px] leading-[1.2] font-bold max-[760px]:text-2xl">
            {film?.title ?? '…'}
          </h1>

          <div className="mb-4 flex flex-wrap gap-2">
            {film?.age_rating ? (
              <span
                className="rounded-full border-transparent px-2.5 py-0.75 text-xs font-bold"
                style={ratingVars(film.age_rating)}
              >
                {film.age_rating}
              </span>
            ) : null}
            {film?.genre ? (
              <span
                className={`rounded-full border px-2.5 py-0.75 text-xs ${INK_BORDER_25} ${INK_85}`}
              >
                {film.genre}
              </span>
            ) : null}
            {film?.duration ? (
              <span
                className={`rounded-full border px-2.5 py-0.75 text-xs ${INK_BORDER_25} ${INK_85}`}
              >
                {formatDuration(film.duration)}
              </span>
            ) : null}
          </div>

          {film?.description ? (
            <p className={`mb-5 leading-relaxed ${INK_80}`}>{film.description}</p>
          ) : null}

          <div className="mb-7 grid grid-cols-[repeat(auto-fit,minmax(160px,1fr))] gap-x-6 gap-y-3">
            {film?.director ? (
              <div>
                <span className={`mb-0.5 block text-xs ${INK_55}`}>{t('movie.director')}</span>
                {film.director}
              </div>
            ) : null}
            {film?.cast ? (
              <div>
                <span className={`mb-0.5 block text-xs ${INK_55}`}>{t('movie.cast')}</span>
                {film.cast}
              </div>
            ) : null}
          </div>

          {seatCheckError ? <Notice variant="error">{seatCheckError}</Notice> : null}

          {film?.status === 'coming_soon' ? (
            // Unstable schedule: no day strip, only the expected date + early shows if any (see ComingSoonHero). `showtimes` is the same hook call above, not a second fetch.
            <ComingSoonHero
              releaseDate={film.release_date}
              showtimes={showtimes.data}
              isFetchingShowtimes={showtimes.isFetching}
              showtimesError={showtimes.error}
              onSelectSeat={handleSelectSeat}
            />
          ) : (
            <>
              <h2 className="mt-0 mb-4 text-xl font-bold">{t('customer.pickShowtime')}</h2>

              <div
                className="mb-4.5 flex gap-2 overflow-x-auto pb-1.5"
                role="group"
                aria-label={t('customer.pickDay')}
              >
                {days.map((day) => {
                  const key = day.format(API_DATE_FORMAT);
                  const active = key === selectedDay;
                  return (
                    <button
                      key={key}
                      type="button"
                      aria-pressed={active}
                      className={[
                        'min-w-17 flex-none rounded-[10px] border px-3 py-2 text-center leading-[1.3]',
                        'transition-[background-color,border-color,color] duration-fast ease-out',
                        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand',
                        active
                          ? 'border-transparent bg-brand font-bold text-on-brand'
                          : `bg-transparent ${INK} ${INK_BORDER_22}`,
                      ].join(' ')}
                      onClick={() => setSelectedDay(key)}
                    >
                      <span className="block text-[11px] opacity-75">{day.format('ddd')}</span>
                      <span className="block text-base font-semibold">{day.format('DD/MM')}</span>
                    </button>
                  );
                })}
              </div>

              {showtimes.error ? (
                <Notice variant="error">
                  {errorMessage(showtimes.error, t('common.somethingWrong'))}
                </Notice>
              ) : null}

              {showtimes.isFetching ? (
                <p className={`text-sm ${INK_62}`}>{t('common.loading')}</p>
              ) : null}

              {!showtimes.isFetching && (showtimes.data?.length ?? 0) === 0 && !showtimes.error ? (
                // Empty has many causes (left `showing`, closed, past, under-priced hall); backend doesn't distinguish, so one honest generic line.
                <EmptyState>{t('customer.noShowtimes')}</EmptyState>
              ) : null}

              {(showtimes.data?.length ?? 0) > 0 ? (
                <div className="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3">
                  {showtimes.data?.map((show) => (
                    <button
                      key={show.id}
                      type="button"
                      className={showtimeCardClass}
                      onClick={() => handleSelectSeat(show)}
                    >
                      <span className={showtimeTimeClass}>
                        {toCinemaTime(show.start_at).format('HH:mm')}
                      </span>
                      <span className={showtimeHallClass}>{show.hall_name}</span>
                      {/* from_price is omitempty: absent at 0 (unpriced hall). Never render "0 d". */}
                      {show.from_price ? (
                        <span className={showtimePriceClass}>
                          {t('customer.fromPrice', { price: formatVND(show.from_price) })}
                        </span>
                      ) : null}
                    </button>
                  ))}
                </div>
              ) : null}
            </>
          )}
        </div>
      </div>

      {sheetOpen ? (
        <LoginBottomSheet
          contextMessage={contextMessage}
          onClose={closeSheet}
          onSuccess={handleSheetSuccess}
        />
      ) : null}
    </div>
  );
};

export default FilmPage;
