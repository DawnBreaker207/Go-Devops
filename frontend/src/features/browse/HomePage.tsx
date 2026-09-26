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

/** Customer home, following QVisionShow frame 1-101: full-bleed hero, a "Now Showing" grid of wide
 *  3:2 cards with a "Show more", and a Trailers section.
 *
 *  The route carries `handle.fullBleed` so the hero can reach the viewport edges; that removes
 *  <main>'s max-width AND padding, so every section below wraps itself in `HOME_SECTION`.
 *
 *  Kept against the frame, deliberately: the Now/Coming/Special tabs. The frame has a plain
 *  "Now Showing" heading, but the tabs are this page's information architecture and the standing
 *  decision is that the Figma is the visual reference, not the IA - dropping them would delete a
 *  working feature to match a picture. The frame's "View All ->" rides in the tab bar instead. */

/** Backend max is 100; this cinema never has that many movies. */
const PAGE_SIZE = 100;

/** Cards shown before "Show more", and how many each press adds. The frame shows 8 (4 x 2). */
const PAGE_STEP = 8;

export const HomePage = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useNowShowing({ page: 1, page_size: PAGE_SIZE });
  const [tab, setTab] = useState<BrowseTab>('showing');
  const [visible, setVisible] = useState(PAGE_STEP);
  const [bookingMovie, setBookingMovie] = useState<Movie | null>(null);

  // GET /movies returns everything incl. draft/ended; customers see only showing + coming_soon.
  const showing = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'showing'),
    [data]
  );
  const comingSoon = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'coming_soon'),
    [data]
  );

  // Switching tabs collapses back to the first page; otherwise "Show more" on a long tab would
  // silently expand a short one. Done in the handler, not an effect: the reset is caused by the
  // click, so there is no external state to synchronise with.
  const changeTab = (next: BrowseTab) => {
    setTab(next);
    setVisible(PAGE_STEP);
  };

  // "Special" is an empty placeholder tab (no such backend concept; UI first, unwired).
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

      {/* pt-16 + the teaser's pt-10 give the frame's ~115px gap between the hero and this section. */}
      <section className={`${HOME_SECTION} pt-16`} aria-labelledby="home-now-showing">
        {/* Follows the tab: a fixed "Dang chieu" would mislabel the section whenever the viewer is
            looking at Sap chieu or Dac biet. */}
        <h2 id="home-now-showing" className="sr-only">
          {tabLabel}
        </h2>

        <BrowseTabs
          tab={tab}
          onChange={changeTab}
          trailing={
            <Link to={`${PATHS.films}?tab=${tab}`} className={`no-underline ${FOCUS_RING}`}>
              {/* Colour on the child: `.cp-customer a` is unlayered and would paint this brand, but
                  the frame draws "View All" in muted grey. */}
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
