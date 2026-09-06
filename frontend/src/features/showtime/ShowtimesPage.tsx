import { Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';

export const ShowtimesPage = () => {
  const { t } = useTranslation();
  return (
    <>
      <PageHeader title={t('menu.showtimes')} />
      <Empty description={t('common.noData')} />
    </>
  );
};

export default ShowtimesPage;
