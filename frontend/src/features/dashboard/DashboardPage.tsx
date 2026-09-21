import { Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import OverviewTab from './components/OverviewTab';
import AnalyticsTab from './components/AnalyticsTab';
import OperationsTab from './components/OperationsTab';
import StaffOverviewSection from './components/StaffOverviewSection';
import { useHasRole } from '@/hooks/useHasRole';

export const DashboardPage = () => {
  const { t } = useTranslation();
  const isAdmin = useHasRole('admin');

  return (
    <>
      <PageHeader title={t('menu.dashboard')} />

      {isAdmin ? (
        <Tabs
          defaultActiveKey="overview"
          items={[
            { key: 'overview', label: t('dashboard.tabOverview'), children: <OverviewTab /> },
            { key: 'analytics', label: t('dashboard.tabAnalytics'), children: <AnalyticsTab /> },
            { key: 'operations', label: t('dashboard.tabOperations'), children: <OperationsTab /> },
          ]}
        />
      ) : (
        <StaffOverviewSection />
      )}
    </>
  );
};

export default DashboardPage;
