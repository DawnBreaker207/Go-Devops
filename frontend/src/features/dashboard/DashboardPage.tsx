import { Card, Col, Row, Statistic } from 'antd';
import {
  CalendarOutlined,
  ScheduleOutlined,
  TeamOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import { useMovieList } from '@/features/movie/hooks/useMovies';

export const DashboardPage = () => {
  const { t } = useTranslation();
  const { data, isFetching } = useMovieList({ page: 1, page_size: 1 });

  const cards = [
    { title: t('menu.movies'), value: data?.meta.total ?? 0, icon: <VideoCameraOutlined /> },
    { title: t('menu.showtimes'), value: 0, icon: <CalendarOutlined /> },
    { title: t('menu.bookings'), value: 0, icon: <ScheduleOutlined /> },
    { title: 'Users', value: 0, icon: <TeamOutlined /> },
  ];

  return (
    <>
      <PageHeader title={t('menu.dashboard')} />
      <Row gutter={[16, 16]}>
        {cards.map((card) => (
          <Col key={card.title} xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title={card.title}
                value={card.value}
                prefix={card.icon}
                loading={isFetching}
              />
            </Card>
          </Col>
        ))}
      </Row>
    </>
  );
};

export default DashboardPage;
