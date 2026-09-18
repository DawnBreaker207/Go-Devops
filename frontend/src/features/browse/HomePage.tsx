import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import PosterCard from './components/PosterCard';
import PosterGridSkeleton from './components/PosterGridSkeleton';
import { useNowShowing } from './hooks/useBrowse';
import { errorMessage } from '@/utils/error';

/**
 * Trang chu khu khach: luoi poster "Dang chieu".
 *
 * Figma frame Home (`77-855`): tieu de can giua, luoi 4 poster mot hang, ten
 * phim can giua ben duoi.
 */

/** Tran cua backend la 100. Rap nay khong co nhieu phim den the. */
const PAGE_SIZE = 100;

export const HomePage = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useNowShowing({ page: 1, page_size: PAGE_SIZE });

  // GET /movies tra ve MOI phim, ke ca `draft` va `ended` - no khong loc gi ca.
  // Khach chi duoc thay phim dang chieu, va do cung la dieu kien de suat chieu
  // cua phim do hien ra o o chon suat.
  const showing = useMemo(
    () => (data?.items ?? []).filter((movie) => movie.status === 'showing'),
    [data]
  );

  return (
    <>
      <h1 className="cp-title" style={{ textAlign: 'center' }}>
        {t('customer.nowShowing')}
      </h1>

      {error ? (
        <div className="cp-notice cp-notice--error" role="alert">
          {errorMessage(error, t('common.somethingWrong'))}
        </div>
      ) : null}

      {isLoading ? <PosterGridSkeleton /> : null}

      {!isLoading && showing.length === 0 && !error ? (
        <p className="cp-empty">{t('customer.noMovies')}</p>
      ) : null}

      {showing.length > 0 ? (
        <div className="cp-poster-grid">
          {showing.map((movie) => (
            <PosterCard key={movie.id} movie={movie} />
          ))}
        </div>
      ) : null}
    </>
  );
};

export default HomePage;
