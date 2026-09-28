import { Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import BoxOfficePage from '@/features/staff/BoxOfficePage';
import CustomerLookupPage from '@/features/staff/CustomerLookupPage';

/** Counter group: Box Office/Customer Lookup tabbed under one sider entry (ROLES_OPERATOR). */
export const CounterPage = () => {
  const { t } = useTranslation();

  return (
    <>
      <PageHeader title={t('menu.counter')} />
      <Tabs
        defaultActiveKey="boxOffice"
        items={[
          { key: 'boxOffice', label: t('menu.boxOffice'), children: <BoxOfficePage /> },
          {
            key: 'customerLookup',
            label: t('menu.customerLookup'),
            children: <CustomerLookupPage />,
          },
        ]}
      />
    </>
  );
};

export default CounterPage;
