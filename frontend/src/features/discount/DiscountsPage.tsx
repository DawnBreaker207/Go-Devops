import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Input,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import DiscountFormModal from './components/DiscountFormModal';
import {
  useCreateDiscount,
  useDeleteDiscount,
  useDiscountList,
  useUpdateDiscount,
} from './hooks/useDiscounts';
import type { CreateDiscountPayload, DiscountCode, UpdateDiscountPayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';

/** Backend takes a Go pointer, so "no filter" must OMIT the field, not send false. */
type ActiveFilter = 'all' | 'active' | 'inactive';

/** Discount codes a customer applies at checkout.
 *
 *  ADMIN-ONLY, unlike the concession catalogue next door: a code moves revenue,
 *  so it is a pricing decision rather than counter work. The router enforces it
 *  with ROLES_ADMIN; this comment is why. */
export const DiscountsPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [activeFilter, setActiveFilter] = useState<ActiveFilter>('all');
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<DiscountCode | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const { data, isFetching, error } = useDiscountList({
    ...query,
    active: activeFilter === 'all' ? undefined : activeFilter === 'active',
  });

  const createDiscount = useCreateDiscount();
  const updateDiscount = useUpdateDiscount();
  const deleteDiscount = useDeleteDiscount();

  // Don't catch here: the modal needs the raw error to bind it to the inputs.
  const handleSubmit = async (payload: CreateDiscountPayload | UpdateDiscountPayload) => {
    if (editing) {
      await updateDiscount.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createDiscount.mutateAsync(payload as CreateDiscountPayload);
      message.success(t('discount.createSuccess'));
    }
    setFormOpen(false);
    setEditing(null);
  };

  const toggleActive = async (row: DiscountCode, next: boolean) => {
    setPendingId(row.id);
    try {
      await updateDiscount.mutateAsync({ id: row.id, payload: { active: next } });
      message.success(t('common.updateSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const handleDelete = async (row: DiscountCode) => {
    setPendingId(row.id);
    try {
      await deleteDiscount.mutateAsync(row.id);
      message.success(t('discount.deleteSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const columns: ColumnsType<DiscountCode> = [
    {
      title: t('discount.code'),
      key: 'code',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong style={{ fontFamily: 'monospace' }}>
            {record.code}
          </Typography.Text>
          {record.description ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.description}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('discount.rule'),
      key: 'rule',
      width: 230,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>
            {record.kind === 'percent'
              ? t('discount.rulePercent', { value: record.value })
              : t('discount.ruleAmount', { value: formatVND(record.value) })}
          </span>
          {/* The cap and the minimum are the two rules that most often explain a
              "why did this code not apply" question, so they are on the row. */}
          {record.kind === 'percent' && record.max_discount ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {t('discount.capUpTo', { value: formatVND(record.max_discount) })}
            </Typography.Text>
          ) : null}
          {record.min_order > 0 ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {t('discount.minFrom', { value: formatVND(record.min_order) })}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('discount.usage'),
      key: 'usage',
      width: 140,
      align: 'right',
      render: (_, record) => {
        const exhausted = record.max_uses != null && record.used_count >= record.max_uses;
        return (
          <Space direction="vertical" size={0} style={{ alignItems: 'flex-end' }}>
            <span className="tabular-nums">
              {record.max_uses == null
                ? t('discount.usedUnlimited', { used: record.used_count })
                : `${record.used_count}/${record.max_uses}`}
            </span>
            {exhausted ? <Tag color="warning">{t('discount.exhausted')}</Tag> : null}
          </Space>
        );
      },
    },
    {
      title: t('discount.window'),
      key: 'window',
      width: 210,
      render: (_, record) =>
        record.starts_at || record.ends_at ? (
          <Typography.Text style={{ fontSize: 12 }}>
            {`${record.starts_at ? formatDateTime(record.starts_at) : '—'} → ${
              record.ends_at ? formatDateTime(record.ends_at) : '—'
            }`}
          </Typography.Text>
        ) : (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('discount.windowAlways')}
          </Typography.Text>
        ),
    },
    {
      title: t('discount.status'),
      dataIndex: 'active',
      key: 'active',
      width: 150,
      render: (value: boolean, record) => (
        <Space size={8}>
          <Switch
            size="small"
            checked={value}
            disabled={pendingId === record.id}
            aria-label={`active-${record.id}`}
            onChange={(next) => void toggleActive(record, next)}
          />
          <Typography.Text type={value ? undefined : 'secondary'}>
            {t(value ? 'discount.enabled' : 'discount.disabled')}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 130,
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="text"
            icon={<EditOutlined />}
            aria-label={`edit-${record.id}`}
            onClick={() => {
              setEditing(record);
              setFormOpen(true);
            }}
          />
          <Popconfirm
            title={t('discount.deleteConfirm')}
            description={t('discount.deleteHint')}
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
        </Space>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title={t('discount.title')}
        extra={
          <Space>
            <Input.Search
              allowClear
              defaultValue={search}
              placeholder={t('discount.searchPlaceholder')}
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

      <Space wrap style={{ marginBottom: 16 }}>
        <Select<ActiveFilter>
          style={{ width: 200 }}
          value={activeFilter}
          onChange={(value) => {
            setActiveFilter(value);
            setPage(1);
          }}
          options={[
            { value: 'all', label: t('discount.filterAll') },
            { value: 'active', label: t('discount.enabled') },
            { value: 'inactive', label: t('discount.disabled') },
          ]}
        />
      </Space>

      {/* The one thing an operator must understand: this really takes money off. */}
      <Alert
        type="warning"
        showIcon
        closable
        style={{ marginBottom: 16 }}
        message={t('discount.realMoneyHint')}
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
        <Table<DiscountCode>
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
            onChange: (nextPage, nextSize) => setPage(nextPage, nextSize),
          }}
        />
      </TableCard>

      <DiscountFormModal
        open={formOpen}
        editing={editing}
        confirmLoading={createDiscount.isPending || updateDiscount.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />
    </>
  );
};

export default DiscountsPage;
