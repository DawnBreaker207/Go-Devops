import { useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import MovieCardWide from './components/MovieCardWide';
import WideGridSkeleton from './components/WideGridSkeleton';
import BrowseTabs, { type BrowseTab } from './components/BrowseTabs';
import BookingModal from './components/BookingModal';
import { useNowShowing } from './hooks/useBrowse';
import { errorMessage } from '@/utils/error';
import type { Movie } from '@/types';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import Panel from '@/components/ui/Panel';
import SectionHead from '@/components/ui/SectionHead';
import { HOME_GRID } from '@/theme/customerTw';

const PAGE_SIZE = 100;

const isBrowseTab = (value: string | null): value is BrowseTab =>
  value === 'showing' || value === 'coming_soon' || value === 'special';

/** Full "Movies" page: same 3 tabs as home but unlimited (home caps at 8/tab). Linked from header nav and home "View more". */
export const FilmsPage = () => {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const initialTab = searchParams.get('tab');
  const [tab, setTab] = useState<BrowseTab>(isBrowseTab(initialTab) ? initialTab : 'showing');
  const [bookingMovie, setBookingMovie] = useState<Movie | null>(null);

  const { data, isLoading, error } = useNowShowing({ page: 1, page_size: PAGE_SIZE });

  const showing = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'showing'),
    [data]
  );
  const comingSoon = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'coming_soon'),
    [data]
  );
  const visible = tab === 'showing' ? showing : tab === 'coming_soon' ? comingSoon : [];

  const changeTab = (next: BrowseTab) => {
    setTab(next);
    setSearchParams(next === 'showing' ? {} : { tab: next }, { replace: true });
  };

  return (
    <>
      <SectionHead title={t('customer.navMovies')} />

      <Panel>
        <BrowseTabs tab={tab} onChange={changeTab} />

        {error ? (
          <Notice variant="error">{errorMessage(error, t('common.somethingWrong'))}</Notice>
        ) : null}

        {isLoading && tab !== 'special' ? <WideGridSkeleton count={12} /> : null}

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
          <div className={HOME_GRID}>
            {visible.map((movie, i) => (
              <MovieCardWide
                key={movie.id}
                movie={movie}
                onBook={movie.status === 'showing' ? setBookingMovie : undefined}
                eager={i < 4}
              />
            ))}
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

export default FilmsPage;
