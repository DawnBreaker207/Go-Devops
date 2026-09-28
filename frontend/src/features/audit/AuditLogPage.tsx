import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  DatePicker,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type dayjs from 'dayjs';
import TableCard from '@/components/TableCard';
import { useAuditLogList } from './hooks/useAuditLogs';
import { diffAuditRows, type AuditDiffRow } from './auditDiff';
import { useListQuery } from '@/hooks/useListQuery';
import { useDebouncedValue } from '@/hooks/useDebouncedValue';
import { AUDIT_ACTIONS, AUDIT_RESOURCE_TYPES, type AuditLog, type AuditOutcome } from '@/types';
import { errorMessage } from '@/utils/error';
import { API_DATE_FORMAT, DATE_FORMAT, formatDateTime } from '@/utils/format';

const OUTCOME_COLOR: Record<AuditOutcome, string> = { success: 'green', failure: 'red' };

const DIFF_ROWS_COLLAPSED = 8;

// Map backend English sentences to audit.err_* keys; keep the original sentence when unmapped.
const ERROR_MESSAGE_SLUGS: Record<string, string> = {
  'nothing to update': 'nothing_to_update',
  'price can not be negative': 'price_negative',
  'end_time must be after start_time': 'invalid_time_range',
  'a percentage discount must be between 1 and 100': 'discount_percent_range',
  'max_discount only applies to a percentage code': 'max_discount_invalid',
  'ends_at must be after starts_at': 'discount_dates_invalid',
  'nothing to update: send active and/or role': 'user_nothing_to_update',
  'phone must be 8 to 15 digits, optionally prefixed with +': 'invalid_phone',
  'unknown template': 'unknown_template',
  'this order belongs to another user': 'forbidden_order',
};

// booking_id threads one order's lifecycle across many resources, so it gets its own filter.
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
  const [showAllDiff, setShowAllDiff] = useState(false);

  const openDetail = (record: AuditLog) => {
    setShowAllDiff(false);
    setDetail(record);
  };

  const diffRows: AuditDiffRow[] = useMemo(
    () => diffAuditRows(detail?.before_json, detail?.after_json),
    [detail]
  );
  const visibleDiffRows = showAllDiff ? diffRows : diffRows.slice(0, DIFF_ROWS_COLLAPSED);

  const translatedErrorMessage = (message?: string): string | undefined => {
    if (!message) return undefined;
    const slug = ERROR_MESSAGE_SLUGS[message.trim()];
    if (!slug) return message;
    return t(`audit.err_${slug}`, { defaultValue: message });
  };

  const actionLabel = (action: string): string =>
    t(`audit.action_${action.replace(/\./g, '_')}`, { defaultValue: action });

  const diffColumns: ColumnsType<AuditDiffRow> = [
    { title: t('audit.field'), dataIndex: 'label', key: 'label', width: 180 },
    { title: t('audit.before'), dataIndex: 'before', key: 'before' },
    { title: t('audit.after'), dataIndex: 'after', key: 'after' },
  ];

  // Debounce the 3 free-text ID boxes; exact-match Selects need no debounce.
  const debouncedResourceId = useDebouncedValue(resourceId);
  const debouncedBookingId = useDebouncedValue(bookingId);
  const debouncedActorId = useDebouncedValue(actorId);

  const { data, isFetching, error } = useAuditLogList({
    ...query,
    action: action || undefined,
    resource_type: resourceType || undefined,
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
      width: 200,
      ellipsis: true,
      render: (value: string) => actionLabel(value),
    },
    {
      title: t('audit.resource'),
      key: 'resource',
      width: 160,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{record.resource_type}</span>
          {record.resource_id ? (
            <Typography.Text
              type="secondary"
              style={{ fontSize: 12 }}
              copyable={{ text: record.resource_id }}
            >
              {record.resource_id.slice(0, 8)}
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
      render: (_, record) => <a onClick={() => openDetail(record)}>{t('audit.viewDetail')}</a>,
    },
  ];

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
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

        <Select
          allowClear
          showSearch
          optionFilterProp="label"
          style={{ width: 240 }}
          placeholder={t('audit.action')}
          value={action || undefined}
          onChange={(value) => {
            setAction(value ?? '');
            resetToFirstPage();
          }}
          options={AUDIT_ACTIONS.map((value) => ({ value, label: actionLabel(value) }))}
        />
        <Select
          allowClear
          showSearch
          style={{ width: 170 }}
          placeholder={t('audit.resourceType')}
          value={resourceType || undefined}
          onChange={(value) => {
            setResourceType(value ?? '');
            resetToFirstPage();
          }}
          options={AUDIT_RESOURCE_TYPES.map((value) => ({ value, label: value }))}
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
        destroyOnHidden
      >
        {detail ? (
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {detail.error_message ? (
              <Alert type="error" showIcon message={translatedErrorMessage(detail.error_message)} />
            ) : null}
            {diffRows.length === 0 ? (
              <Typography.Text type="secondary">{t('audit.noChanges')}</Typography.Text>
            ) : (
              <>
                <Table<AuditDiffRow>
                  rowKey="key"
                  size="small"
                  pagination={false}
                  columns={diffColumns}
                  dataSource={visibleDiffRows}
                />
                {!showAllDiff && diffRows.length > DIFF_ROWS_COLLAPSED ? (
                  <Button type="link" style={{ padding: 0 }} onClick={() => setShowAllDiff(true)}>
                    {t('audit.showMore')}
                  </Button>
                ) : null}
              </>
            )}
            {detail.ip || detail.user_agent ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {detail.ip} · {detail.user_agent}
              </Typography.Text>
            ) : null}
            <Typography.Text
              type="secondary"
              copyable={{ text: detail.id }}
              style={{ fontSize: 12 }}
            >
              ID: {detail.id}
            </Typography.Text>
          </Space>
        ) : null}
      </Modal>
    </>
  );
};

export default AuditLogPage;
