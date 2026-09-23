import { useTranslation } from 'react-i18next';
import StaticPage from '@/components/ui/StaticPage';
import SectionHead from '@/components/ui/SectionHead';
import EmptyState from '@/components/ui/EmptyState';

/** Static "Offers" page. No offers source in the backend, so show an honest empty state instead of fake data. */
export const OffersPage = () => {
  const { t } = useTranslation();

  return (
    <StaticPage>
      <SectionHead title={t('customer.offersTitle')} />
      <EmptyState>{t('customer.offersEmpty')}</EmptyState>
    </StaticPage>
  );
};

export default OffersPage;
