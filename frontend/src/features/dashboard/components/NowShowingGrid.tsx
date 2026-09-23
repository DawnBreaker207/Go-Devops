import { Alert, Col, Empty, Row, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import MovieCard, { MovieCardSkeleton } from '@/features/movie/components/MovieCard';
import { useMovieList } from '@/features/movie/hooks/useMovies';
import { safeMessage } from '@/utils/error';

const PREVIEW_SIZE = 6;

/** Now-showing film grid (status=showing only - the unfiltered list mixes in coming_soon). Lives at the end of the Overview tab only. */
export const NowShowingGrid = () => {
  const { t } = useTranslation();
  const { data, isFetching, error } = useMovieList({
    page: 1,
    page_size: PREVIEW_SIZE,
    status: 'showing',
  });
  const movies = data?.items ?? [];

  return (
    <>
      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={safeMessage(error, t('common.somethingWrong'))}
        />
      ) : null}
      <Typography.Title level={5} style={{ marginTop: 24, marginBottom: 12 }}>
        {t('movie.nowShowing')}
      </Typography.Title>
      {/* No above-the-fold entrance animation (never pre-hide with opacity 0), only card micro-interaction + skeleton. */}
      {!isFetching && movies.length === 0 ? (
        <Empty />
      ) : (
        <Row gutter={[16, 16]}>
          {(isFetching ? Array.from({ length: PREVIEW_SIZE }) : movies).map((movie, index) => (
            <Col key={movie ? (movie as (typeof movies)[number]).id : index} xs={12} sm={8} lg={4}>
              {movie ? (
                <MovieCard movie={movie as (typeof movies)[number]} />
              ) : (
                <MovieCardSkeleton />
              )}
            </Col>
          ))}
        </Row>
      )}
    </>
  );
};

export default NowShowingGrid;
