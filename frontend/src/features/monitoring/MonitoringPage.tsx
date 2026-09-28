import { Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import AuditLogPage from '@/features/audit/AuditLogPage';
import BatchJobsPage from '@/features/batch/BatchJobsPage';

/** Monitoring group: Audit Logs/Batch Jobs tabbed under one sider entry (ROLES_ADMIN).
 *  Business reporting lives as a Dashboard tab instead - these two are technical/ops tools,
 *  a different audience from a revenue report. */
export const MonitoringPage = () => {
  const { t } = useTranslation();

  return (
    <>
      <PageHeader title={t('menu.monitoring')} />
      <Tabs
        defaultActiveKey="auditLogs"
        items={[
          { key: 'auditLogs', label: t('menu.auditLogs'), children: <AuditLogPage /> },
          { key: 'batchJobs', label: t('menu.batchJobs'), children: <BatchJobsPage /> },
        ]}
      />
    </>
  );
};

export default MonitoringPage;
