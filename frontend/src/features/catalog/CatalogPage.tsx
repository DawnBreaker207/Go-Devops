import { Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import MoviesPage from '@/features/movie/MoviesPage';
import ShowtimesPage from '@/features/showtime/ShowtimesPage';
import HallsPage from '@/features/hall/HallsPage';
import ConcessionsPage from '@/features/concession/ConcessionsPage';

/** Catalog group: Movies/Showtimes/Halls/Concessions tabbed under one sider entry (ROLES_OPERATOR). */
export const CatalogPage = () => {
  const { t } = useTranslation();

  return (
    <>
      <PageHeader title={t('menu.catalog')} />
      <Tabs
        defaultActiveKey="movies"
        items={[
          { key: 'movies', label: t('menu.movies'), children: <MoviesPage /> },
          { key: 'halls', label: t('menu.halls'), children: <HallsPage /> },
          { key: 'showtimes', label: t('menu.showtimes'), children: <ShowtimesPage /> },
          { key: 'concessions', label: t('menu.concessions'), children: <ConcessionsPage /> },
        ]}
      />
    </>
  );
};

export default CatalogPage;
