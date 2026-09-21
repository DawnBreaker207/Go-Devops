import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import PosterCard from './components/PosterCard';
import PosterGridSkeleton from './components/PosterGridSkeleton';
import BannerCarousel from './components/BannerCarousel';
import BrowseTabs, { type BrowseTab } from './components/BrowseTabs';
import UpcomingTicketTeaser from './components/UpcomingTicketTeaser';
import BookingModal from './components/BookingModal';
import { useNowShowing } from './hooks/useBrowse';
import { errorMessage } from '@/utils/error';
import { PATHS } from '@/routes/paths';
import type { Movie } from '@/types';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import Panel from '@/components/ui/Panel';
import { BROWSE_GRID, INK_65, INK_BORDER_20, TRANSITION_FAST } from '@/theme/customerTw';

/** Customer home: banner + Now/Coming tabs + poster grid (Figma Home 77-855). */

/** Backend max is 100; this cinema never has that many movies. */
const PAGE_SIZE = 100;

/** Max movies per tab before "View more". */
const HOME_LIMIT = 8;

export const HomePage = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useNowShowing({ page: 1, page_size: PAGE_SIZE });
  const [tab, setTab] = useState<BrowseTab>('showing');
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
  // "Special" is an empty placeholder tab (no such backend concept; UI first, unwired).
  const allForTab = tab === 'showing' ? showing : tab === 'coming_soon' ? comingSoon : [];
  const visible = allForTab.slice(0, HOME_LIMIT);
  const hasMore = allForTab.length > HOME_LIMIT;

  return (
    <>
      <BannerCarousel movies={showing} onBook={setBookingMovie} />

      <UpcomingTicketTeaser />
      {/* Straight into the tab+grid card, no centered title block (duplicates the tab) and no cinema pill (single cinema). Search lives in the header, shared by all pages. */}
      <Panel>
        <BrowseTabs tab={tab} onChange={setTab} />

        {error ? (
          <Notice variant="error">{errorMessage(error, t('common.somethingWrong'))}</Notice>
        ) : null}

        {isLoading && tab !== 'special' ? <PosterGridSkeleton /> : null}

        {!isLoading && visible.length === 0 && !error ? (
          <EmptyState>
            {tab === 'showing'
              ? t('customer.noMovies')
              : tab === 'coming_soon'
                ? t('customer.noComingSoon')
                : t('customer.noSpecial')}
          </EmptyState>
        ) : null}

        {visible.length > 0 ? (
          <div className={BROWSE_GRID}>
            {visible.map((movie) => (
              <PosterCard key={movie.id} movie={movie} onBook={setBookingMovie} />
            ))}

            {hasMore ? (
              <Link
                to={`${PATHS.films}?tab=${tab}`}
                className={`flex h-full min-h-40 flex-col items-center justify-center gap-1 rounded-xl border border-dashed p-4 text-center text-sm font-semibold no-underline ${INK_BORDER_20} ${INK_65} ${TRANSITION_FAST} hover-fine:border-brand hover-fine:text-brand`}
              >
                <span aria-hidden="true" className="text-2xl">
                  →
                </span>
                {t('customer.viewMore')}
              </Link>
            ) : null}
          </div>
        ) : null}
      </Panel>

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
