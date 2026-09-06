import { Button, Result } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PATHS } from '@/routes/paths';

export const NotFoundPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Result
      status="404"
      title={t('error.notFoundTitle')}
      subTitle={t('error.notFoundSubtitle')}
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
      }}
      extra={
        <Button type="primary" onClick={() => navigate(PATHS.dashboard)}>
          {t('error.backHome')}
        </Button>
      }
    />
  );
};

export default NotFoundPage;
