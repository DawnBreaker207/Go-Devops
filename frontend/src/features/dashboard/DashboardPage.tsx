import { Alert, Card, Col, Empty, Row, Statistic, Typography } from 'antd';
import {
  CalendarOutlined,
  ScheduleOutlined,
  TeamOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import MovieCard, { MovieCardSkeleton } from '@/features/movie/components/MovieCard';
import { useMovieList } from '@/features/movie/hooks/useMovies';
import { useAdminStats } from './hooks/useDashboard';
import { useHasRole } from '@/hooks/useHasRole';
import { formatNumber } from '@/utils/format';
import { errorMessage } from '@/utils/error';

const PREVIEW_SIZE = 6;

export const DashboardPage = () => {
  const { t } = useTranslation();

  // GET /admin/stats la admin-only. Staff van dung dashboard nay, chi la khong
  // co o so lieu - tot hon la hien bon o luon bao 0 nhu truoc.
  const isAdmin = useHasRole('admin');
  const { data: stats, isFetching: statsLoading, error: statsError } = useAdminStats(isAdmin);

  const { data, isFetching } = useMovieList({ page: 1, page_size: PREVIEW_SIZE });

  const cards = [
    { key: 'movies', title: t('menu.movies'), value: stats?.movies, icon: <VideoCameraOutlined /> },
    {
      key: 'showtimes',
      title: t('menu.showtimes'),
      value: stats?.showtimes,
      icon: <CalendarOutlined />,
    },
    {
      key: 'bookings',
      title: t('dashboard.confirmedBookings'),
      value: stats?.bookings,
      icon: <ScheduleOutlined />,
    },
    { key: 'users', title: t('menu.users'), value: stats?.users, icon: <TeamOutlined /> },
  ];

  const movies = data?.items ?? [];

  return (
    <>
      <PageHeader title={t('menu.dashboard')} />

      {isAdmin ? (
        <>
          {statsError ? (
            <Alert
              type="error"
              showIcon
              style={{ marginBottom: 16 }}
              message={errorMessage(statsError, t('common.somethingWrong'))}
            />
          ) : null}
          <Row gutter={[16, 16]}>
            {cards.map((card) => (
              <Col key={card.key} xs={24} sm={12} lg={6}>
                <Card>
                  <Statistic
                    title={card.title}
                    value={card.value === undefined ? '—' : formatNumber(card.value)}
                    prefix={card.icon}
                    loading={statsLoading}
                  />
                </Card>
              </Col>
            ))}
          </Row>
        </>
      ) : null}

      <Typography.Title level={5} style={{ marginTop: isAdmin ? 24 : 0, marginBottom: 12 }}>
        {t('movie.nowShowing')}
      </Typography.Title>

      {/* Noi dung nay nam tren fold nen KHONG co animation luc vao man: rule cam
       * render san opacity 0 cho noi dung chinh. Chuyen dong o day chi la
       * micro-interaction cua tung card (hover/focus/press) va skeleton -> that. */}
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

export default DashboardPage;
