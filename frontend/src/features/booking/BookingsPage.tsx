import { Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';

export const BookingsPage = () => {
  const { t } = useTranslation();
  return (
    <>
      <PageHeader title={t('menu.bookings')} />
      <Empty description={t('common.noData')} />
    </>
  );
};

export default BookingsPage;
