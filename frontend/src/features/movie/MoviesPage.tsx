import { useState } from 'react';
import { App, Button, Input, Popconfirm, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import MovieFormModal from './components/MovieFormModal';
import { useCreateMovie, useDeleteMovie, useMovieList, useUpdateMovie } from './hooks/useMovies';
import type { ApiError, Movie, MoviePayload, MovieStatus } from '@/types';
import { formatDate, formatDuration } from '@/utils/format';

const STATUS_COLOR: Record<MovieStatus, string> = {
  draft: 'default',
  showing: 'green',
  ended: 'red',
};

export const MoviesPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Movie | null>(null);

  const { data, isFetching } = useMovieList({ page, page_size: pageSize, search });
  const createMovie = useCreateMovie();
  const updateMovie = useUpdateMovie();
  const deleteMovie = useDeleteMovie();

  const openCreate = () => {
    setEditing(null);
    setModalOpen(true);
  };

  const openEdit = (movie: Movie) => {
    setEditing(movie);
    setModalOpen(true);
  };

  const handleSubmit = async (payload: MoviePayload) => {
    try {
      if (editing) {
        await updateMovie.mutateAsync({ id: editing.id, payload });
        message.success(t('common.updateSuccess'));
      } else {
        await createMovie.mutateAsync(payload);
        message.success(t('common.createSuccess'));
      }
      setModalOpen(false);
      setEditing(null);
    } catch (error) {
      message.error((error as ApiError).message || t('common.somethingWrong'));
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteMovie.mutateAsync(id);
      message.success(t('common.deleteSuccess'));
    } catch (error) {
      message.error((error as ApiError).message || t('common.somethingWrong'));
    }
  };

  const columns: ColumnsType<Movie> = [
    { title: t('movie.name'), dataIndex: 'title', key: 'title', ellipsis: true },
    { title: t('movie.genre'), dataIndex: 'genre', key: 'genre', width: 140 },
    {
      title: t('movie.duration'),
      dataIndex: 'duration',
      key: 'duration',
      width: 110,
      render: (value: number) => formatDuration(value),
    },
    { title: t('movie.director'), dataIndex: 'director', key: 'director', width: 180 },
    {
      title: t('movie.releaseDate'),
      dataIndex: 'release_date',
      key: 'release_date',
      width: 140,
      render: (value: string) => formatDate(value),
    },
    {
      title: t('movie.status'),
      dataIndex: 'status',
      key: 'status',
      width: 130,
      render: (status: MovieStatus) => (
        <Tag color={STATUS_COLOR[status]}>
          {t(`movie.status${status.charAt(0).toUpperCase()}${status.slice(1)}`)}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 120,
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
            title={t('common.deleteConfirm')}
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

  return (
    <>
      <PageHeader
        title={t('movie.title')}
        extra={
          <Space>
            <Input.Search
              allowClear
              placeholder={t('movie.searchPlaceholder')}
              style={{ width: 280 }}
              onSearch={(value) => {
                setSearch(value);
                setPage(1);
              }}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              {t('common.create')}
            </Button>
          </Space>
        }
      />

      <Table<Movie>
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
          showTotal: (total) => `${total}`,
          onChange: (nextPage, nextSize) => {
            setPage(nextPage);
            setPageSize(nextSize);
          },
        }}
      />

      <MovieFormModal
        open={modalOpen}
        movie={editing}
        confirmLoading={createMovie.isPending || updateMovie.isPending}
        onCancel={() => {
          setModalOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />
    </>
  );
};

export default MoviesPage;
