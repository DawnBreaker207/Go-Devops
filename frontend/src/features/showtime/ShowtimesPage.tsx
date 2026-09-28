import { useState } from 'react';
import {
  App,
  Alert,
  Button,
  DatePicker,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, PlusOutlined, StopOutlined } from '@ant-design/icons';
import type dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import TableCard from '@/components/TableCard';
import ShowtimeFormModal from './components/ShowtimeFormModal';
import {
  useCreateShowtime,
  useCancelShowtime,
  useDeleteShowtime,
  useHallOptions,
  useMovieOptions,
  useShowtimeList,
  useUpdateShowtime,
} from './hooks/useShowtimes';
import type { Showtime, ShowtimePayload, ShowtimeStatus } from '@/types';
import { SHOWTIME_STATUSES } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { API_DATE_FORMAT, DATE_FORMAT, formatDateTime, toCinemaTime } from '@/utils/format';

const STATUS_COLOR: Record<ShowtimeStatus, string> = {
  open: 'green',
  closed: 'default',
  // Cancelled is a refund, not a quiet close - never the default color.
  cancelled: 'red',
};

// A map, not a ternary: a ternary once rendered CANCELLED as closed.
const STATUS_LABEL_KEY: Record<ShowtimeStatus, string> = {
  open: 'showtime.statusOpen',
  closed: 'showtime.statusClosed',
  cancelled: 'showtime.statusCancelled',
};

const pastStyle: React.CSSProperties = { opacity: 0.45 };

export const ShowtimesPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, setPage } = useListQuery();
  const [movieId, setMovieId] = useState<string>();
  const [hallId, setHallId] = useState<string>();
  const [status, setStatus] = useState<ShowtimeStatus>();
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Showtime | null>(null);

  const movies = useMovieOptions();
  const halls = useHallOptions();

  const { data, isFetching, error } = useShowtimeList({
    ...query,
    movie_id: movieId,
    hall_id: hallId,
    status,
    from: range?.[0]?.format(API_DATE_FORMAT),
    to: range?.[1]?.format(API_DATE_FORMAT),
    // Nearest first; the backend defaults to start_at DESC.
    sort: 'start_at',
    order: 'asc',
  });

  const createShowtime = useCreateShowtime();
  const updateShowtime = useUpdateShowtime();
  const deleteShowtime = useDeleteShowtime();
  const cancelShowtime = useCancelShowtime();

  const openCreate = () => {
    setEditing(null);
    setModalOpen(true);
  };

  const openEdit = (showtime: Showtime) => {
    setEditing(showtime);
    setModalOpen(true);
  };

  // Don't catch: the modal binds errors onto inputs itself.
  const handleSubmit = async (payload: ShowtimePayload) => {
    if (editing) {
      await updateShowtime.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createShowtime.mutateAsync(payload);
      message.success(t('common.createSuccess'));
    }
    setModalOpen(false);
    setEditing(null);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteShowtime.mutateAsync(id);
      message.success(t('common.deleteSuccess'));
    } catch (err) {
      // Every conflict shares 40900; tell apart by the backend sentence, never swallow it.
      message.error(errorMessage(err, t('common.somethingWrong')));
    }
  };

  // Cancel is a refund; DELETE 409s on any booking, so this is the only way to stop a sold showtime.
  const handleCancel = async (id: string) => {
    try {
      const result = await cancelShowtime.mutateAsync(id);
      message.success(
        result.bookings_affected > 0
          ? t('showtime.cancelSuccessWithRefunds', { count: result.bookings_affected })
          : t('showtime.cancelSuccess')
      );
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    }
  };

  const columns: ColumnsType<Showtime> = [
    {
      title: t('showtime.movie'),
      dataIndex: 'movie_title',
      key: 'movie_title',
      ellipsis: true,
      render: (title: string, record) => (
        <Space size={6}>
          <span>{title}</span>
          {record.age_rating ? <Tag bordered={false}>{record.age_rating}</Tag> : null}
        </Space>
      ),
    },
    { title: t('showtime.hall'), dataIndex: 'hall_name', key: 'hall_name', width: 170 },
    {
      title: t('showtime.startAt'),
      dataIndex: 'start_at',
      key: 'start_at',
      width: 170,
      // Past shows stay visible but dimmed so sellable ones scan first.
      render: (value: string) => (
        <span style={toCinemaTime(value).isBefore(Date.now()) ? pastStyle : undefined}>
          {formatDateTime(value)}
        </span>
      ),
    },
    {
      title: t('showtime.endAt'),
      dataIndex: 'end_at',
      key: 'end_at',
      width: 170,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('showtime.status'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value: ShowtimeStatus) => (
        <Tag color={STATUS_COLOR[value]} bordered={false}>
          {t(STATUS_LABEL_KEY[value])}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 150,
      align: 'right',
      render: (_, record) => {
        const done = record.status === 'cancelled';
        return (
          <Space>
            <Tooltip title={t('common.edit')}>
              <Button
                type="text"
                icon={<EditOutlined />}
                aria-label={`edit-${record.id}`}
                disabled={done}
                onClick={() => openEdit(record)}
              />
            </Tooltip>
            <Tooltip title={t('showtime.cancelCta')}>
              <Popconfirm
                title={t('showtime.cancelConfirm')}
                description={t('showtime.cancelHint')}
                okText={t('showtime.cancelCta')}
                cancelText={t('common.cancel')}
                okButtonProps={{ danger: true, loading: cancelShowtime.isPending }}
                onConfirm={() => handleCancel(record.id)}
              >
                <Button
                  danger
                  type="text"
                  icon={<StopOutlined />}
                  aria-label={`cancel-${record.id}`}
                  disabled={done}
                />
              </Popconfirm>
            </Tooltip>
            <Tooltip title={t('common.delete')}>
              <Popconfirm
                title={t('showtime.deleteConfirm')}
                description={t('showtime.deleteHint')}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
                onConfirm={() => handleDelete(record.id)}
              >
                <Button
                  danger
                  type="text"
                  icon={<DeleteOutlined />}
                  aria-label={`delete-${record.id}`}
                  disabled={done}
                />
              </Popconfirm>
            </Tooltip>
          </Space>
        );
      },
    },
  ];

  const resetToFirstPage = () => setPage(1);

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('common.create')}
        </Button>
        <Select
          allowClear
          showSearch
          optionFilterProp="label"
          style={{ width: 220 }}
          placeholder={t('showtime.filterMovie')}
          value={movieId}
          onChange={(value) => {
            setMovieId(value);
            resetToFirstPage();
          }}
          options={(movies.data?.items ?? []).map((m) => ({ value: m.id, label: m.title }))}
        />
        <Select
          allowClear
          style={{ width: 190 }}
          placeholder={t('showtime.filterHall')}
          value={hallId}
          onChange={(value) => {
            setHallId(value);
            resetToFirstPage();
          }}
          options={(halls.data?.items ?? []).map((h) => ({ value: h.id, label: h.name }))}
        />
        <Select<ShowtimeStatus>
          allowClear
          style={{ width: 150 }}
          placeholder={t('showtime.filterStatus')}
          value={status}
          onChange={(value) => {
            setStatus(value);
            resetToFirstPage();
          }}
          options={[
            ...SHOWTIME_STATUSES.map((value) => ({
              value,
              label: t(STATUS_LABEL_KEY[value]),
            })),
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
        <Table<Showtime>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
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
      </TableCard>

      <ShowtimeFormModal
        open={modalOpen}
        showtime={editing}
        confirmLoading={createShowtime.isPending || updateShowtime.isPending}
        onCancel={() => {
          setModalOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />
    </>
  );
};

export default ShowtimesPage;
