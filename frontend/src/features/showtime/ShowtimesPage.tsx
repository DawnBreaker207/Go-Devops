import { useState } from 'react';
import { App, Alert, Button, DatePicker, Popconfirm, Select, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import type dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import ShowtimeFormModal from './components/ShowtimeFormModal';
import {
  useCreateShowtime,
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
};

const pastStyle: React.CSSProperties = { opacity: 0.45 };

export const ShowtimesPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, setPage } = useListQuery();
  // Bo loc rieng cua man nay: khong nhet vao useListQuery vi helper do chi lo
  // page/page_size/search dung chung cho moi bang.
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
    // Suat chieu doc theo thoi gian nen sap tang dan de gan nhat len truoc;
    // mac dinh cua backend la start_at DESC.
    sort: 'start_at',
    order: 'asc',
  });

  const createShowtime = useCreateShowtime();
  const updateShowtime = useUpdateShowtime();
  const deleteShowtime = useDeleteShowtime();

  const openCreate = () => {
    setEditing(null);
    setModalOpen(true);
  };

  const openEdit = (showtime: Showtime) => {
    setEditing(showtime);
    setModalOpen(true);
  };

  // Khong bat loi o day: modal can chinh loi de gan details vao dung o nhap.
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
      // Moi xung dot cua suat chieu deu la 40900, phan biet bang message cua
      // backend - no da la cau tieng Anh doc duoc, dung nuot di.
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
      // Suat da chieu xong van phai hien (day la danh sach van hanh), nhung lam
      // mo di de nguoi truc doc luot la biet cai nao con ban duoc.
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
          {t(`showtime.status${value === 'open' ? 'Open' : 'Closed'}`)}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 110,
      align: 'right',
      render: (_, record) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            aria-label={`edit-${record.id}`}
            onClick={() => openEdit(record)}
          />
          <Popconfirm
            title={t('showtime.deleteConfirm')}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button
              danger
              type="text"
              icon={<DeleteOutlined />}
              aria-label={`delete-${record.id}`}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const resetToFirstPage = () => setPage(1);

  return (
    <>
      <PageHeader
        title={t('showtime.title')}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            {t('common.create')}
          </Button>
        }
      />

      <Space wrap style={{ marginBottom: 16 }}>
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
              label: t(`showtime.status${value === 'open' ? 'Open' : 'Closed'}`),
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

      {/* Loi cua chinh danh sach chan ca man hinh nen dung Alert, khong dung toast. */}
      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

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
