import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import MovieCardWide from './components/MovieCardWide';
import WideGridSkeleton from './components/WideGridSkeleton';
import BannerCarousel from './components/BannerCarousel';
import BrowseTabs, { type BrowseTab } from './components/BrowseTabs';
import TrailersSection from './components/TrailersSection';
import UpcomingTicketTeaser from './components/UpcomingTicketTeaser';
import BookingModal from './components/BookingModal';
import { useNowShowing } from './hooks/useBrowse';
import { errorMessage } from '@/utils/error';
import { PATHS } from '@/routes/paths';
import type { Movie } from '@/types';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import Button from '@/components/ui/Button';
import { FOCUS_RING, HOME_GRID, HOME_SECTION, INK_65, TRANSITION_FAST } from '@/theme/customerTw';

/** Backend max is 100; this cinema never has that many movies. */
const PAGE_SIZE = 100;

const PAGE_STEP = 8;

export const HomePage = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useNowShowing({ page: 1, page_size: PAGE_SIZE });
  const [tab, setTab] = useState<BrowseTab>('showing');
  const [visible, setVisible] = useState(PAGE_STEP);
  const [bookingMovie, setBookingMovie] = useState<Movie | null>(null);

  // GET /movies returns drafts/ended too; filter to showing + coming_soon for customers.
  const showing = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'showing'),
    [data]
  );
  const comingSoon = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'coming_soon'),
    [data]
  );

  // Reset to the first page on tab switch, or "Show more" on a long tab leaks into a short one.
  const changeTab = (next: BrowseTab) => {
    setTab(next);
    setVisible(PAGE_STEP);
  };

  // The Special tab is an unwired placeholder (no backend concept).
  const allForTab = tab === 'showing' ? showing : tab === 'coming_soon' ? comingSoon : [];
  const tabLabel =
    tab === 'showing'
      ? t('customer.nowShowing')
      : tab === 'coming_soon'
        ? t('customer.comingSoon')
        : t('customer.special');
  const shown = allForTab.slice(0, visible);
  const hasMore = allForTab.length > visible;

  return (
    <>
      <BannerCarousel movies={showing} onBook={setBookingMovie} />

      <div className={`${HOME_SECTION} pt-10`}>
        <UpcomingTicketTeaser />
      </div>

      <section className={`${HOME_SECTION} pt-16`} aria-labelledby="home-now-showing">
        <h2 id="home-now-showing" className="sr-only">
          {tabLabel}
        </h2>

        <BrowseTabs
          tab={tab}
          onChange={changeTab}
          trailing={
            <Link to={`${PATHS.films}?tab=${tab}`} className={`no-underline ${FOCUS_RING}`}>
              <span
                className={`inline-flex items-center gap-2 text-sm font-semibold ${INK_65} ${TRANSITION_FAST} hover-fine:text-brand`}
              >
                {t('customer.viewAll')}
                <span aria-hidden="true">→</span>
              </span>
            </Link>
          }
        />

        {error ? (
          <Notice variant="error">{errorMessage(error, t('common.somethingWrong'))}</Notice>
        ) : null}

        {isLoading && tab !== 'special' ? <WideGridSkeleton /> : null}

        {!isLoading && shown.length === 0 && !error ? (
          <EmptyState>
            {tab === 'showing'
              ? t('customer.noMovies')
              : tab === 'coming_soon'
                ? t('customer.noComingSoon')
                : t('customer.noSpecial')}
          </EmptyState>
        ) : null}

        {shown.length > 0 ? (
          <>
            <div className={HOME_GRID}>
              {shown.map((movie, i) => (
                <MovieCardWide
                  key={movie.id}
                  movie={movie}
                  onBook={movie.status === 'showing' ? setBookingMovie : undefined}
                  eager={i < 4}
                />
              ))}
            </div>

            {hasMore ? (
              <div className="mt-10 flex justify-center">
                <Button variant="primary" onClick={() => setVisible((n) => n + PAGE_STEP)}>
                  {t('customer.showMore')}
                </Button>
              </div>
            ) : null}
          </>
        ) : null}
      </section>

      <TrailersSection movies={showing} />

      <div className="pb-12 max-[900px]:pb-[calc(48px+var(--cp-bottomnav-height))]" />

      {bookingMovie ? (
        <BookingModal
          movieId={bookingMovie.id}
          movieTitle={bookingMovie.title}
          onClose={() => setBookingMovie(null)}
        />
      ) : null}
    </>
  );
};

export default HomePage;
