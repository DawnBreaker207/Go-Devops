import { useTranslation } from 'react-i18next';
import StaticPage, { StaticPageBody } from '@/components/ui/StaticPage';
import SectionHead from '@/components/ui/SectionHead';

/** Static "Cinema" page for the bottom nav. Single cinema: no picker/GPS, just a short static page. */
export const CinemaInfoPage = () => {
  const { t } = useTranslation();

  return (
    <StaticPage>
      <SectionHead
        eyebrow={t('customer.nowShowingEyebrow')}
        title={t('customer.cinemaInfoTitle')}
      />
      <StaticPageBody>{t('customer.cinemaInfoBody')}</StaticPageBody>
    </StaticPage>
  );
};

export default CinemaInfoPage;
