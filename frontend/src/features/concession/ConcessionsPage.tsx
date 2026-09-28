import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Input,
  Popconfirm,
  Select,
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
import TableCard from '@/components/TableCard';
import ConcessionFormModal from './components/ConcessionFormModal';
import {
  useConcessionList,
  useCreateConcession,
  useDeleteConcession,
  useUpdateConcession,
} from './hooks/useConcessions';
import type { Combo, CreateComboPayload, UpdateComboPayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';

// The backend takes a Go pointer, so "no filter" must OMIT the field rather than send false.
type ActiveFilter = 'all' | 'active' | 'inactive';

// Concession stock for step 2; admin + staff, since restocking is counter work.
export const ConcessionsPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [activeFilter, setActiveFilter] = useState<ActiveFilter>('all');
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Combo | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const { data, isFetching, error } = useConcessionList({
    ...query,
    active: activeFilter === 'all' ? undefined : activeFilter === 'active',
  });

  const createConcession = useCreateConcession();
  const updateConcession = useUpdateConcession();
  const deleteConcession = useDeleteConcession();

  // Don't catch here: the modal binds 400/40001 onto inputs.
  const handleSubmit = async (payload: CreateComboPayload | UpdateComboPayload) => {
    if (editing) {
      await updateConcession.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createConcession.mutateAsync(payload as CreateComboPayload);
      message.success(t('concession.createSuccess'));
    }
    setFormOpen(false);
    setEditing(null);
  };

  // The inline switch sends ONLY `active` - that's the whole point of PATCH.
  const toggleActive = async (row: Combo, next: boolean) => {
    setPendingId(row.id);
    try {
      await updateConcession.mutateAsync({ id: row.id, payload: { active: next } });
      message.success(t('common.updateSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const handleDelete = async (row: Combo) => {
    setPendingId(row.id);
    try {
      await deleteConcession.mutateAsync(row.id);
      message.success(t('concession.deleteSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (row: Combo) => {
    setEditing(row);
    setFormOpen(true);
  };

  const columns: ColumnsType<Combo> = [
    {
      title: t('concession.product'),
      key: 'product',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{record.name}</span>
          {record.description ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.description}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('concession.price'),
      dataIndex: 'price',
      key: 'price',
      width: 150,
      align: 'right',
      // 0 is a real, chosen price (a giveaway), never "unset"; say so rather than showing a bare "0 d".
      render: (value: number) =>
        value === 0 ? <Tag color="warning">{t('concession.priceFree')}</Tag> : formatVND(value),
    },
    {
      title: t('concession.status'),
      dataIndex: 'active',
      key: 'active',
      width: 130,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'} bordered={false}>
          {t(value ? 'concession.onSale' : 'concession.offSale')}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 170,
      render: (_, record) => (
        <Space size={4}>
          <Tooltip title={record.active ? t('concession.markOffSale') : t('concession.markOnSale')}>
            <Popconfirm
              title={record.active ? t('concession.offSaleConfirm') : t('concession.onSaleConfirm')}
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
              onClick={() => openEdit(record)}
            />
          </Tooltip>
          <Tooltip title={t('common.delete')}>
            <Popconfirm
              title={t('concession.deleteConfirm')}
              description={t('concession.deleteHint')}
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
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          defaultValue={search}
          placeholder={t('concession.searchPlaceholder')}
          style={{ width: 300 }}
          onSearch={setSearch}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('common.create')}
        </Button>
        <Select<ActiveFilter>
          style={{ width: 200 }}
          value={activeFilter}
          onChange={(value) => {
            setActiveFilter(value);
            setPage(1);
          }}
          options={[
            { value: 'all', label: t('concession.filterAll') },
            { value: 'active', label: t('concession.onSale') },
            { value: 'inactive', label: t('concession.offSale') },
          ]}
        />
      </Space>

      {/* Key point for operators: this list differs from what a customer sees. */}
      <Alert
        type="info"
        showIcon
        closable
        style={{ marginBottom: 16 }}
        message={t('concession.scopeHint')}
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
        <Table<Combo>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          scroll={{ x: 800 }}
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

      <ConcessionFormModal
        open={formOpen}
        editing={editing}
        confirmLoading={createConcession.isPending || updateConcession.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />
    </>
  );
};

export default ConcessionsPage;
