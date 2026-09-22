import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Input,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import UserFormModal from './components/UserFormModal';
import { useCreateUser, useUpdateUser, useUserList } from './hooks/useUsers';
import type { CreateUserPayload, User, UserRole } from '@/types';
import { USER_ROLES } from '@/types';
import { useAuthStore } from '@/stores/authStore';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { formatDateTime } from '@/utils/format';

/** `active` filter: backend takes a pointer, so "no filter" must OMIT the field, unlike false. */
type ActiveFilter = 'all' | 'active' | 'locked';

export const UsersPage = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const currentUserId = useAuthStore((s) => s.user?.id);

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [role, setRole] = useState<UserRole>();
  const [activeFilter, setActiveFilter] = useState<ActiveFilter>('all');
  const [formOpen, setFormOpen] = useState(false);
  /** Row awaiting the server, so only that row locks. */
  const [pendingId, setPendingId] = useState<string | null>(null);

  const { data, isFetching, error } = useUserList({
    ...query,
    role,
    active: activeFilter === 'all' ? undefined : activeFilter === 'active',
  });

  const createUser = useCreateUser();
  const updateUser = useUpdateUser();

  // Don't catch here: the modal needs the raw error for input binding.
  const handleCreate = async (payload: CreateUserPayload) => {
    await createUser.mutateAsync(payload);
    message.success(t('user.createSuccess'));
    setFormOpen(false);
  };

  const applyChange = async (user: User, patch: { active?: boolean; role?: UserRole }) => {
    setPendingId(user.id);
    try {
      await updateUser.mutateAsync({ id: user.id, payload: patch });
      message.success(t('common.updateSuccess'));
    } catch (err) {
      // All four backend guards share 409/40900 and differ by sentence only (self-lock, self-demote, last active admin). Keep the server sentence instead of guessing which.
      message.error(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setPendingId(null);
    }
  };

  const resetToFirstPage = () => setPage(1);

  const columns: ColumnsType<User> = [
    {
      title: t('user.account'),
      key: 'account',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <Space size={6}>
            <span>{record.full_name || '-'}</span>
            {record.id === currentUserId ? <Tag bordered={false}>{t('user.you')}</Tag> : null}
          </Space>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {record.email}
            {record.phone ? ` · ${record.phone}` : ''}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: t('user.role'),
      dataIndex: 'role',
      key: 'role',
      width: 190,
      render: (value: UserRole, record) => {
        // Backend 409s self role changes; disable upfront instead of serving an error.
        const isSelf = record.id === currentUserId;
        const select = (
          <Select<UserRole>
            size="small"
            style={{ width: 150 }}
            value={value}
            disabled={isSelf || pendingId === record.id}
            aria-label={`role-${record.id}`}
            onChange={(next) => void applyChange(record, { role: next })}
            options={USER_ROLES.map((r) => ({ value: r, label: t(`user.role_${r}`) }))}
          />
        );
        return isSelf ? <Tooltip title={t('user.cannotChangeOwnRole')}>{select}</Tooltip> : select;
      },
    },
    {
      title: t('user.status'),
      dataIndex: 'active',
      key: 'active',
      width: 150,
      render: (value: boolean, record) => {
        // Only self-lock is blocked; self-unlock can't happen (locked accounts can't sign in).
        const isSelf = record.id === currentUserId;
        const control = (
          <Space size={8}>
            <Switch
              size="small"
              checked={value}
              disabled={(isSelf && value) || pendingId === record.id}
              aria-label={`active-${record.id}`}
              onChange={(next) => void applyChange(record, { active: next })}
            />
            <Typography.Text type={value ? undefined : 'secondary'}>
              {t(value ? 'user.statusActive' : 'user.statusLocked')}
            </Typography.Text>
          </Space>
        );
        return isSelf && value ? (
          <Tooltip title={t('user.cannotLockSelf')}>{control}</Tooltip>
        ) : (
          control
        );
      },
    },
    {
      title: t('user.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
  ];

  return (
    <>
      <PageHeader
        title={t('user.title')}
        extra={
          <Space>
            <Input.Search
              allowClear
              defaultValue={search}
              placeholder={t('user.searchPlaceholder')}
              style={{ width: 300 }}
              onSearch={setSearch}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
              {t('common.create')}
            </Button>
          </Space>
        }
      />

      <Space wrap style={{ marginBottom: 16 }}>
        <Select<UserRole>
          allowClear
          style={{ width: 170 }}
          placeholder={t('user.role')}
          value={role}
          onChange={(value) => {
            setRole(value);
            resetToFirstPage();
          }}
          options={USER_ROLES.map((r) => ({ value: r, label: t(`user.role_${r}`) }))}
        />
        <Select<ActiveFilter>
          style={{ width: 190 }}
          value={activeFilter}
          onChange={(value) => {
            setActiveFilter(value);
            resetToFirstPage();
          }}
          options={[
            { value: 'all', label: t('user.filterAll') },
            { value: 'active', label: t('user.statusActive') },
            { value: 'locked', label: t('user.statusLocked') },
          ]}
        />
      </Space>

      <Alert
        type="info"
        showIcon
        closable
        style={{ marginBottom: 16 }}
        message={t('user.guardsHint')}
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
        <Table<User>
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

      <UserFormModal
        open={formOpen}
        confirmLoading={createUser.isPending}
        onCancel={() => setFormOpen(false)}
        onSubmit={handleCreate}
      />
    </>
  );
};

export default UsersPage;
