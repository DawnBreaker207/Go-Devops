import { useState } from 'react';
import { Alert, DatePicker, Input, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type dayjs from 'dayjs';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import { useAuditLogList } from './hooks/useAuditLogs';
import { useListQuery } from '@/hooks/useListQuery';
import { useDebouncedValue } from '@/hooks/useDebouncedValue';
import type { AuditLog, AuditOutcome } from '@/types';
import { errorMessage } from '@/utils/error';
import { API_DATE_FORMAT, DATE_FORMAT, formatDateTime } from '@/utils/format';

const OUTCOME_COLOR: Record<AuditOutcome, string> = { success: 'green', failure: 'red' };

/** Newest-first event log. `booking_id` threads one order's lifecycle across differing resource_type/resource_id, so it is a standalone filter. */
export const AuditLogPage = () => {
  const { t } = useTranslation();
  const { query, page, pageSize, setPage } = useListQuery();

  const [action, setAction] = useState('');
  const [resourceType, setResourceType] = useState('');
  const [resourceId, setResourceId] = useState('');
  const [bookingId, setBookingId] = useState('');
  const [actorId, setActorId] = useState('');
  const [outcome, setOutcome] = useState<AuditOutcome>();
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [detail, setDetail] = useState<AuditLog | null>(null);

  // Debounce the 5 free-text filters; each keystroke used to fire a request.
  const debouncedAction = useDebouncedValue(action);
  const debouncedResourceType = useDebouncedValue(resourceType);
  const debouncedResourceId = useDebouncedValue(resourceId);
  const debouncedBookingId = useDebouncedValue(bookingId);
  const debouncedActorId = useDebouncedValue(actorId);

  const { data, isFetching, error } = useAuditLogList({
    ...query,
    action: debouncedAction || undefined,
    resource_type: debouncedResourceType || undefined,
    resource_id: debouncedResourceId || undefined,
    booking_id: debouncedBookingId || undefined,
    actor_id: debouncedActorId || undefined,
    outcome,
    from: range?.[0]?.format(API_DATE_FORMAT),
    to: range?.[1]?.format(API_DATE_FORMAT),
  });

  const resetToFirstPage = () => setPage(1);

  const columns: ColumnsType<AuditLog> = [
    {
      title: t('audit.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('audit.action'),
      dataIndex: 'action',
      key: 'action',
      width: 180,
      ellipsis: true,
    },
    {
      title: t('audit.resource'),
      key: 'resource',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{record.resource_type}</span>
          {record.resource_id ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }} copyable>
              {record.resource_id}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('audit.bookingId'),
      dataIndex: 'booking_id',
      key: 'booking_id',
      width: 140,
      ellipsis: true,
      render: (value?: string) =>
        value ? <Typography.Text copyable>{value}</Typography.Text> : '-',
    },
    {
      title: t('audit.actor'),
      key: 'actor',
      width: 160,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{record.actor_role || '-'}</span>
          {record.actor_id ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }} ellipsis>
              {record.actor_id}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('audit.outcome'),
      dataIndex: 'outcome',
      key: 'outcome',
      width: 110,
      render: (value: AuditOutcome) => (
        <Tag color={OUTCOME_COLOR[value]} bordered={false}>
          {t(`audit.outcome_${value}`)}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 100,
      align: 'right',
      render: (_, record) => <a onClick={() => setDetail(record)}>{t('audit.viewDetail')}</a>,
    },
  ];

  return (
    <>
      <PageHeader title={t('audit.title')} />

      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('audit.action')}
          style={{ width: 180 }}
          value={action}
          onChange={(e) => {
            setAction(e.target.value);
            resetToFirstPage();
          }}
        />
        <Input
          placeholder={t('audit.resourceType')}
          style={{ width: 160 }}
          value={resourceType}
          onChange={(e) => {
            setResourceType(e.target.value);
            resetToFirstPage();
          }}
        />
        <Input
          placeholder={t('audit.resourceId')}
          style={{ width: 180 }}
          value={resourceId}
          onChange={(e) => {
            setResourceId(e.target.value);
            resetToFirstPage();
          }}
        />
        <Input
          placeholder={t('audit.bookingId')}
          style={{ width: 180 }}
          value={bookingId}
          onChange={(e) => {
            setBookingId(e.target.value);
            resetToFirstPage();
          }}
        />
        <Input
          placeholder={t('audit.actor')}
          style={{ width: 180 }}
          value={actorId}
          onChange={(e) => {
            setActorId(e.target.value);
            resetToFirstPage();
          }}
        />
        <Select<AuditOutcome>
          allowClear
          style={{ width: 150 }}
          placeholder={t('audit.outcome')}
          value={outcome}
          onChange={(value) => {
            setOutcome(value);
            resetToFirstPage();
          }}
          options={[
            { value: 'success', label: t('audit.outcome_success') },
            { value: 'failure', label: t('audit.outcome_failure') },
          ]}
        />
        <DatePicker.RangePicker
          format={DATE_FORMAT}
          value={range}
          onChange={(value) => {
            setRange(value as [dayjs.Dayjs, dayjs.Dayjs] | null);
            resetToFirstPage();
          }}
        />
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
        <Table<AuditLog>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          scroll={{ x: 1300 }}
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

      <Modal
        open={Boolean(detail)}
        onCancel={() => setDetail(null)}
        onOk={() => setDetail(null)}
        title={t('audit.viewDetail')}
        width={720}
      >
        {detail ? (
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {detail.error_message ? (
              <Alert type="error" showIcon message={detail.error_message} />
            ) : null}
            <div>
              <Typography.Text strong>{t('audit.before')}</Typography.Text>
              <pre style={{ maxHeight: 240, overflow: 'auto', margin: 0 }}>
                {JSON.stringify(detail.before_json ?? null, null, 2)}
              </pre>
            </div>
            <div>
              <Typography.Text strong>{t('audit.after')}</Typography.Text>
              <pre style={{ maxHeight: 240, overflow: 'auto', margin: 0 }}>
                {JSON.stringify(detail.after_json ?? null, null, 2)}
              </pre>
            </div>
            {detail.ip || detail.user_agent ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {detail.ip} · {detail.user_agent}
              </Typography.Text>
            ) : null}
          </Space>
        ) : null}
      </Modal>
    </>
  );
};

export default AuditLogPage;
