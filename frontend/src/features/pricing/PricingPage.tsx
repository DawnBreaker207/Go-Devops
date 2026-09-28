import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Divider,
  Input,
  Popconfirm,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  StopOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import BasePriceForm from './components/BasePriceForm';
import PricingRuleFormModal from './components/PricingRuleFormModal';
import {
  useCreatePricingRule,
  useDeletePricingRule,
  usePricingRules,
  useUpdatePricingRule,
} from './hooks/usePricing';
import type { CreatePricingRulePayload, PricingRule, UpdatePricingRulePayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { formatDate } from '@/utils/format';

/** Global pricing: base seat-type prices (inline form) + adjustment rules (table). ADMIN-ONLY
 *  (ROLES_ADMIN): a rule moves revenue, same reasoning as discounts. */
export const PricingPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<PricingRule | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const { data, isFetching, error } = usePricingRules(query);

  const createRule = useCreatePricingRule();
  const updateRule = useUpdatePricingRule();
  const deleteRule = useDeletePricingRule();

  // Don't catch here: the modal needs the raw error to bind it to the inputs.
  const handleSubmit = async (payload: CreatePricingRulePayload | UpdatePricingRulePayload) => {
    if (editing) {
      await updateRule.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createRule.mutateAsync(payload as CreatePricingRulePayload);
      message.success(t('pricing.createRuleSuccess'));
    }
    setFormOpen(false);
    setEditing(null);
  };

  const toggleActive = async (row: PricingRule, next: boolean) => {
    setPendingId(row.id);
    try {
      await updateRule.mutateAsync({ id: row.id, payload: { active: next } });
      message.success(t('common.updateSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const handleDelete = async (row: PricingRule) => {
    setPendingId(row.id);
    try {
      await deleteRule.mutateAsync(row.id);
      message.success(t('pricing.deleteRuleSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const columns: ColumnsType<PricingRule> = [
    {
      title: t('pricing.ruleName'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: t('pricing.dayOfWeek'),
      key: 'day_of_week',
      width: 120,
      render: (_, record) =>
        record.day_of_week === undefined ? (
          <Typography.Text type="secondary">{t('pricing.dayOfWeekAny')}</Typography.Text>
        ) : (
          t(`pricing.dow_${record.day_of_week}`)
        ),
    },
    {
      title: t('pricing.timeWindow'),
      key: 'time_window',
      width: 140,
      render: (_, record) =>
        record.start_time || record.end_time ? (
          `${record.start_time?.slice(0, 5) ?? '—'} - ${record.end_time?.slice(0, 5) ?? '—'}`
        ) : (
          <Typography.Text type="secondary">{t('pricing.timeWindowAny')}</Typography.Text>
        ),
    },
    {
      title: t('pricing.specificDate'),
      dataIndex: 'specific_date',
      key: 'specific_date',
      width: 130,
      render: (value: string | undefined) =>
        value ? formatDate(value) : <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: t('pricing.adjustment'),
      key: 'adjustment',
      width: 150,
      render: (_, record) =>
        record.adjust_kind === 'percent'
          ? t('pricing.adjustPercent', { value: record.adjust_value })
          : t('pricing.adjustFixed', { value: record.adjust_value }),
    },
    {
      title: t('pricing.priority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      align: 'right',
    },
    {
      title: t('pricing.status'),
      dataIndex: 'active',
      key: 'active',
      width: 120,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'} bordered={false}>
          {t(value ? 'pricing.enabled' : 'pricing.disabled')}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size={4}>
          <Tooltip title={record.active ? t('pricing.markOff') : t('pricing.markOn')}>
            <Popconfirm
              title={record.active ? t('pricing.offConfirm') : t('pricing.onConfirm')}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              onConfirm={() => void toggleActive(record, !record.active)}
            >
              <Button
                type="text"
                icon={record.active ? <StopOutlined /> : <CheckCircleOutlined />}
                aria-label={`toggle-active-${record.id}`}
                disabled={pendingId === record.id}
                loading={pendingId === record.id}
              />
            </Popconfirm>
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={`edit-${record.id}`}
              onClick={() => {
                setEditing(record);
                setFormOpen(true);
              }}
            />
          </Tooltip>
          <Tooltip title={t('common.delete')}>
            <Popconfirm
              title={t('pricing.deleteRuleConfirm')}
              okText={t('common.delete')}
              cancelText={t('common.cancel')}
              okButtonProps={{ danger: true }}
              onConfirm={() => void handleDelete(record)}
            >
              <Button
                danger
                type="text"
                icon={<DeleteOutlined />}
                aria-label={`delete-${record.id}`}
                disabled={pendingId === record.id}
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <>
      <PageHeader title={t('pricing.title')} />

      <BasePriceForm />

      <Divider />

      <PageHeader
        title={t('pricing.rulesTitle')}
        extra={
          <Space>
            <Input.Search
              allowClear
              defaultValue={search}
              placeholder={t('pricing.searchPlaceholder')}
              style={{ width: 280 }}
              onSearch={setSearch}
            />
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditing(null);
                setFormOpen(true);
              }}
            >
              {t('common.create')}
            </Button>
          </Space>
        }
      />

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      <TableCard>
        <Table<PricingRule>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          scroll={{ x: 1000 }}
          pagination={{
            current: data?.meta.page ?? page,
            pageSize: data?.meta.page_size ?? pageSize,
            total: data?.meta.total ?? 0,
            showSizeChanger: true,
            showTotal: (total) => t('common.totalItems', { total }),
            onChange: (nextPage, nextSize) => setPage(nextPage, nextSize),
          }}
        />
      </TableCard>

      <PricingRuleFormModal
        open={formOpen}
        entity={editing}
        confirmLoading={createRule.isPending || updateRule.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />
    </>
  );
};

export default PricingPage;
