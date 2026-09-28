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
  DeleteOutlined,
  EditOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import TableCard from '@/components/TableCard';
import ArticleFormModal from './components/ArticleFormModal';
import {
  useAdminArticleList,
  useCreateArticle,
  useDeleteArticle,
  useUpdateArticle,
} from './hooks/useArticles';
import type { Article, ArticleType, CreateArticlePayload, UpdateArticlePayload } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { formatDateTime } from '@/utils/format';

type TypeFilter = 'all' | ArticleType;

const statusColor = (status: Article['status']): string => {
  switch (status) {
    case 'published':
      return 'success';
    case 'hidden':
      return 'default';
    default:
      return 'warning';
  }
};

/** Article CMS (news/promotions). Operator scope: admin AND staff, like the concession catalogue -
 *  a news post moves no money, so it's content work, not a pricing decision. Enforced via ROLES_OPERATOR. */
export const ArticlesPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all');
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Article | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);

  const { data, isFetching, error } = useAdminArticleList({
    ...query,
    type: typeFilter === 'all' ? undefined : typeFilter,
  });

  const createArticle = useCreateArticle();
  const updateArticle = useUpdateArticle();
  const deleteArticle = useDeleteArticle();

  // Don't catch here: the modal needs the raw error to bind it to the inputs.
  const handleSubmit = async (payload: CreateArticlePayload | UpdateArticlePayload) => {
    if (editing) {
      await updateArticle.mutateAsync({ id: editing.id, payload });
      message.success(t('common.updateSuccess'));
    } else {
      await createArticle.mutateAsync(payload as CreateArticlePayload);
      message.success(t('article.createSuccess'));
    }
    setFormOpen(false);
    setEditing(null);
  };

  const handleDelete = async (row: Article) => {
    setPendingId(row.id);
    try {
      await deleteArticle.mutateAsync(row.id);
      message.success(t('article.deleteSuccess'));
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  // Publishing/hiding is a quick action, not a form field - the form only sets the initial
  // draft status on create (see ArticleFormModal). A draft or a hidden post can be published
  // straight from here; a published one can only be hidden, not sent back to draft.
  const handleSetStatus = async (row: Article, status: Article['status']) => {
    setPendingId(row.id);
    try {
      await updateArticle.mutateAsync({ id: row.id, payload: { status } });
      message.success(
        t(status === 'published' ? 'article.publishedSuccess' : 'article.hiddenSuccess')
      );
    } catch (err) {
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const columns: ColumnsType<Article> = [
    {
      title: t('article.title'),
      key: 'title',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{record.title}</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            /{record.slug} · {t(`article.type${record.type === 'news' ? 'News' : 'Promotion'}`)}
          </Typography.Text>
          {record.summary ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.summary}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('article.status'),
      dataIndex: 'status',
      key: 'status',
      width: 140,
      render: (value: Article['status']) => (
        <Tag color={statusColor(value)}>
          {t(`article.status${value[0].toUpperCase()}${value.slice(1)}`)}
        </Tag>
      ),
    },
    {
      title: t('article.views'),
      dataIndex: 'views',
      key: 'views',
      width: 100,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{value}</span>,
    },
    {
      title: t('article.updated'),
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (value: string) => (
        <Typography.Text style={{ fontSize: 12 }}>{formatDateTime(value)}</Typography.Text>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 160,
      render: (_, record) => (
        <Space size={4}>
          {record.status === 'published' ? (
            <Tooltip title={t('article.markHidden')}>
              <Popconfirm
                title={t('article.hideConfirm')}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
                onConfirm={() => void handleSetStatus(record, 'hidden')}
              >
                <Button
                  type="text"
                  icon={<EyeInvisibleOutlined />}
                  aria-label={`hide-${record.id}`}
                  disabled={pendingId === record.id}
                  loading={pendingId === record.id}
                />
              </Popconfirm>
            </Tooltip>
          ) : (
            <Tooltip title={t('article.markPublished')}>
              <Popconfirm
                title={t('article.publishConfirm')}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
                onConfirm={() => void handleSetStatus(record, 'published')}
              >
                <Button
                  type="text"
                  icon={<EyeOutlined />}
                  aria-label={`publish-${record.id}`}
                  disabled={pendingId === record.id}
                  loading={pendingId === record.id}
                />
              </Popconfirm>
            </Tooltip>
          )}
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
              title={t('article.deleteConfirm')}
              description={t('article.deleteHint')}
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
          placeholder={t('article.searchPlaceholder')}
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
        <Select<TypeFilter>
          style={{ width: 200 }}
          value={typeFilter}
          onChange={(value) => {
            setTypeFilter(value);
            setPage(1);
          }}
          options={[
            { value: 'all', label: t('article.filterAll') },
            { value: 'news', label: t('article.typeNews') },
            { value: 'promotion', label: t('article.typePromotion') },
          ]}
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
        <Table<Article>
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
            onChange: (nextPage, nextSize) => setPage(nextPage, nextSize),
          }}
        />
      </TableCard>

      <ArticleFormModal
        open={formOpen}
        editing={editing}
        confirmLoading={createArticle.isPending || updateArticle.isPending}
        onCancel={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
      />
    </>
  );
};

export default ArticlesPage;
