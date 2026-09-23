import { App, Alert, Button, Input, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlayCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import { useBatchJobList, useRunBatchJob } from './hooks/useBatchJobs';
import { useListQuery } from '@/hooks/useListQuery';
import type { BatchJob, BatchJobName, BatchJobStatus } from '@/types';
import { BATCH_JOB_NAMES } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatNumber } from '@/utils/format';

const STATUS_COLOR: Record<BatchJobStatus, string> = {
  running: 'blue',
  success: 'green',
  failed: 'red',
  skipped: 'default',
  stopped: 'orange',
};

/** Job-run history + manual run of known jobs. Manual runs return 202 on enqueue; watch the table for the real outcome (it polls via refetchInterval in useBatchJobList). */
export const BatchJobsPage = () => {
  const { t } = useTranslation();
  const { message, modal } = App.useApp();
  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();

  const { data, isFetching, error } = useBatchJobList(query);
  const runJob = useRunBatchJob();

  const runNow = (name: BatchJobName) => {
    modal.confirm({
      title: t('batch.runConfirmTitle', { name }),
      content: t('batch.runConfirmBody'),
      okText: t('batch.runAction'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await runJob.mutateAsync(name);
          message.success(t('batch.runStarted', { name }));
        } catch (err) {
          message.error(errorMessage(err, t('common.somethingWrong')));
        }
      },
    });
  };

  const columns: ColumnsType<BatchJob> = [
    { title: t('batch.jobName'), dataIndex: 'job_name', key: 'job_name', width: 180 },
    {
      title: t('batch.status'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (value: BatchJobStatus) => (
        <Tag color={STATUS_COLOR[value]} bordered={false}>
          {t(`batch.status_${value}`)}
        </Tag>
      ),
    },
    {
      title: t('batch.triggeredBy'),
      dataIndex: 'triggered_by',
      key: 'triggered_by',
      width: 110,
      render: (value: string) => t(`batch.triggeredBy_${value}`, value),
    },
    {
      title: t('batch.processedRows'),
      dataIndex: 'processed_rows',
      key: 'processed_rows',
      width: 110,
      align: 'right',
      render: (value: number) => formatNumber(value),
    },
    {
      title: t('batch.skippedRows'),
      dataIndex: 'skipped_rows',
      key: 'skipped_rows',
      width: 110,
      align: 'right',
      render: (value: number) => formatNumber(value),
    },
    {
      title: t('batch.startedAt'),
      dataIndex: 'started_at',
      key: 'started_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('batch.finishedAt'),
      dataIndex: 'finished_at',
      key: 'finished_at',
      width: 160,
      render: (value?: string) => formatDateTime(value),
    },
    {
      title: t('batch.errorMessage'),
      dataIndex: 'error_message',
      key: 'error_message',
      ellipsis: true,
      render: (value?: string) => value || '-',
    },
  ];

  return (
    <>
      <PageHeader
        title={t('batch.title')}
        extra={
          <Input.Search
            allowClear
            defaultValue={search}
            placeholder={t('batch.searchPlaceholder')}
            style={{ width: 260 }}
            onSearch={setSearch}
          />
        }
      />

      <Space wrap style={{ marginBottom: 16 }}>
        {BATCH_JOB_NAMES.map((name) => (
          <Button
            key={name}
            icon={<PlayCircleOutlined />}
            onClick={() => runNow(name)}
            loading={runJob.isPending && runJob.variables === name}
          >
            {t('batch.runJob', { name })}
          </Button>
        ))}
      </Space>

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      <TableCard>
        <Table<BatchJob>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          scroll={{ x: 1100 }}
          pagination={{
            current: data?.meta.page ?? page,
            pageSize: data?.meta.page_size ?? pageSize,
            total: data?.meta.total ?? 0,
            showSizeChanger: true,
            showTotal: (total) => t('common.totalItems', { total }),
            onChange: setPage,
          }}
        />
      </TableCard>
    </>
  );
};

export default BatchJobsPage;
