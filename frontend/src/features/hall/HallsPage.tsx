import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Alert,
  App,
  Button,
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
  AppstoreOutlined,
  CopyOutlined,
  DeleteOutlined,
  DollarOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import HallFormModal from './components/HallFormModal';
import HallPriceModal from './components/HallPriceModal';
import CloneHallModal from './components/CloneHallModal';
import {
  isPriceSetComplete,
  useCreateHall,
  useDeleteHall,
  useHallList,
  useHallPriceStatuses,
  useUpdateHall,
} from './hooks/useHalls';
import type { Hall, HallPayload, UpdateHallPayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { hallSeatsPath } from '@/routes/paths';
import { errorMessage } from '@/utils/error';

export const HallsPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const { data, isFetching, error } = useHallList(query);

  const halls = data?.items ?? [];
  const priceStatuses = useHallPriceStatuses(halls);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Hall | null>(null);
  const [pricingHall, setPricingHall] = useState<Hall | null>(null);
  const [cloningHall, setCloningHall] = useState<Hall | null>(null);

  const createHall = useCreateHall();
  const updateHall = useUpdateHall();
  const deleteHall = useDeleteHall();

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (hall: Hall) => {
    setEditing(hall);
    setFormOpen(true);
  };

  // Khong bat loi o day: modal can chinh loi de gan details vao dung o nhap.
  const handleCreate = async (payload: HallPayload) => {
    await createHall.mutateAsync(payload);
    message.success(t('common.createSuccess'));
    setFormOpen(false);
  };

  const handleUpdate = async (payload: UpdateHallPayload) => {
    if (!editing) return;
    await updateHall.mutateAsync({ id: editing.id, payload });
    message.success(t('common.updateSuccess'));
    setFormOpen(false);
    setEditing(null);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHall.mutateAsync(id);
      message.success(t('common.deleteSuccess'));
    } catch (err) {
      // 409/40900 khi phong con suat chieu chua chieu xong. Thong bao cua backend
      // la tieng Anh nhung cu the hon bat cu cau nao viet san o day.
      message.error(errorMessage(err, t('common.somethingWrong')));
    }
  };

  const columns: ColumnsType<Hall> = [
    {
      title: t('hall.name'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      render: (name: string, record) => (
        <Space direction="vertical" size={0}>
          <Link to={hallSeatsPath(record.id)}>{name}</Link>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t(`hall.screen_${record.screen_position}`)}
            {record.aisle_after_cols.length > 0
              ? ` · ${t('hall.aislesShort', { cols: record.aisle_after_cols.join(', ') })}`
              : ''}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: t('hall.grid'),
      key: 'grid',
      width: 120,
      align: 'right',
      // rows x seats_per_row la so O LUOI, khong phai suc chua: ghe doi nuot mot
      // cot va ghe is_gap khong ban duoc. Suc chua that nam trong man so do ghe.
      render: (_, record) => (
        <span className="tabular-nums">
          {record.rows} × {record.seats_per_row}
        </span>
      ),
    },
    {
      title: t('hall.prices'),
      key: 'prices',
      width: 190,
      render: (_, record) => {
        const rows = priceStatuses.byHallId[record.id];
        if (rows === undefined) {
          return <Typography.Text type="secondary">…</Typography.Text>;
        }
        if (isPriceSetComplete(rows)) {
          return (
            <Tag color="green" bordered={false}>
              {t('hall.pricesComplete')}
            </Tag>
          );
        }
        return (
          // Day la loi im lang duy nhat backend khong bao: danh sach suat chieu
          // cho khach JOIN hall_prices va HAVING count(DISTINCT seat_type) = 4,
          // nen phong thieu gia bien mat khoi phia khach ma van hien o admin.
          <Tooltip title={t('hall.pricesIncompleteTooltip')}>
            <Tag color="warning" bordered={false}>
              {t('hall.pricesIncomplete', { count: rows.length })}
            </Tag>
          </Tooltip>
        );
      },
    },
    {
      title: t('hall.active'),
      dataIndex: 'active',
      key: 'active',
      width: 120,
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'default'} bordered={false}>
          {t(active ? 'hall.activeYes' : 'hall.activeNo')}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 190,
      align: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Tooltip title={t('hall.seats')}>
            <Link to={hallSeatsPath(record.id)}>
              <Button type="text" icon={<AppstoreOutlined />} aria-label={`seats-${record.id}`} />
            </Link>
          </Tooltip>
          <Tooltip title={t('hall.prices')}>
            <Button
              type="text"
              icon={<DollarOutlined />}
              aria-label={`prices-${record.id}`}
              onClick={() => setPricingHall(record)}
            />
          </Tooltip>
          <Tooltip title={t('hall.clone')}>
            <Button
              type="text"
              icon={<CopyOutlined />}
              aria-label={`clone-${record.id}`}
              onClick={() => setCloningHall(record)}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              icon={<EditOutlined />}
              aria-label={`edit-${record.id}`}
              onClick={() => openEdit(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('hall.deleteConfirm')}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
            onConfirm={() => handleDelete(record.id)}
          >
            {/* Nut xoa trong Figma la nut DAC mau do, khong phai icon vien. */}
            <Button
              danger
              type="primary"
              icon={<DeleteOutlined />}
              aria-label={`delete-${record.id}`}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title={t('hall.title')}
        extra={
          <Space>
            <Input.Search
              allowClear
              defaultValue={search}
              placeholder={t('hall.searchPlaceholder')}
              style={{ width: 280 }}
              onSearch={setSearch}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
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

      <Table<Hall>
        rowKey="id"
        columns={columns}
        dataSource={halls}
        loading={isFetching}
        scroll={{ x: 900 }}
        pagination={{
          current: data?.meta.page ?? page,
          pageSize: data?.meta.page_size ?? pageSize,
          total: data?.meta.total ?? 0,
          showSizeChanger: true,
          showTotal: (total) => t('common.totalItems', { total }),
          onChange: setPage,
        }}
      />

      <HallFormModal
        open={formOpen}
        hall={editing}
        confirmLoading={createHall.isPending || updateHall.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onCreate={handleCreate}
        onUpdate={handleUpdate}
      />

      <HallPriceModal
        open={pricingHall !== null}
        hall={pricingHall}
        onCancel={() => setPricingHall(null)}
        onDone={() => setPricingHall(null)}
      />

      <CloneHallModal
        open={cloningHall !== null}
        hall={cloningHall}
        onCancel={() => setCloningHall(null)}
        onDone={() => setCloningHall(null)}
      />
    </>
  );
};

export default HallsPage;
