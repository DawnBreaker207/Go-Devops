import { useState } from 'react';
import { App, Button, Input, Popconfirm, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, PlayCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import TableCard from '@/components/TableCard';
import MovieFormModal from './components/MovieFormModal';
import MoviePoster from './components/MoviePoster';
import { useCreateMovie, useDeleteMovie, useMovieList, useUpdateMovie } from './hooks/useMovies';
import type { Movie, MoviePayload, MovieStatus } from '@/types';
import { formatDate, formatDuration } from '@/utils/format';
import { errorMessage } from '@/utils/error';
import { useListQuery } from '@/hooks/useListQuery';

const STATUS_COLOR: Record<MovieStatus, string> = {
  draft: 'default',
  coming_soon: 'blue',
  showing: 'green',
  ended: 'red',
};

/** The one-way quick-action path; `ended` and `showing` have no further forward step here -
 *  going back a step (e.g. pulling a movie from `showing`) still goes through Edit. */
const NEXT_STATUS: Partial<Record<MovieStatus, MovieStatus>> = {
  draft: 'coming_soon',
  coming_soon: 'showing',
};

export const MoviesPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  // List state lives in the URL (see useListQuery).
  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Movie | null>(null);

  const { data, isFetching } = useMovieList(query);
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

  // Don't catch here: the modal needs the raw error to bind details to inputs (400/40001). Rethrow is how it learns the save failed.
  const handleSubmit = async (payload: MoviePayload) => {
    if (editing) {
      await updateMovie.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createMovie.mutateAsync(payload);
      message.success(t('common.createSuccess'));
    }
    setModalOpen(false);
    setEditing(null);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteMovie.mutateAsync(id);
      message.success(t('common.deleteSuccess'));
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const statusLabel = (status: MovieStatus) =>
    t(`movie.status${status.charAt(0).toUpperCase()}${status.slice(1)}`);

  // Shortcut to the next lifecycle step (draft -> coming_soon -> showing): full PUT replay of
  // the row's own fields with only `status` changed, since PUT /movies/:id is a full replace.
  const handlePromote = async (movie: Movie, nextStatus: MovieStatus) => {
    try {
      await updateMovie.mutateAsync({
        id: movie.id,
        payload: {
          title: movie.title,
          genre: movie.genre,
          duration: movie.duration,
          director: movie.director,
          description: movie.description,
          poster_url: movie.poster_url,
          backdrop_url: movie.backdrop_url,
          trailer_url: movie.trailer_url,
          cast: movie.cast,
          age_rating: movie.age_rating,
          release_date: movie.release_date,
          status: nextStatus,
        },
      });
      message.success(t('movie.promotedTo', { status: statusLabel(nextStatus) }));
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const columns: ColumnsType<Movie> = [
    {
      title: t('movie.cover'),
      dataIndex: 'poster_url',
      key: 'poster_url',
      width: 88,
      render: (url: string, record) => <MoviePoster url={url} title={record.title} />,
    },
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
      width: 150,
      align: 'right',
      render: (_, record) => {
        const nextStatus = NEXT_STATUS[record.status];
        return (
          <Space>
            {nextStatus ? (
              <Popconfirm
                title={t('movie.promoteConfirm', { status: statusLabel(nextStatus) })}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
                onConfirm={() => void handlePromote(record, nextStatus)}
              >
                <Button
                  type="text"
                  icon={<PlayCircleOutlined />}
                  aria-label={`promote-${record.id}`}
                  title={t('movie.promoteTo', { status: statusLabel(nextStatus) })}
                />
              </Popconfirm>
            ) : null}
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
        );
      },
    },
  ];

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          defaultValue={search}
          placeholder={t('movie.searchPlaceholder')}
          style={{ width: 280 }}
          onSearch={setSearch}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('common.create')}
        </Button>
      </Space>

      <TableCard>
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
            showTotal: (total) => t('common.totalItems', { total }),
            onChange: setPage,
          }}
        />
      </TableCard>

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
