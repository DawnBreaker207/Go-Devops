import { useState } from 'react';
import { Alert, Button, Input, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import CustomerDetailDrawer from './components/CustomerDetailDrawer';
import { useCustomerList } from './hooks/useCustomerLookup';
import { useListQuery } from '@/hooks/useListQuery';
import type { User } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime } from '@/utils/format';

/** Staff customer lookup: GET /staff/customers returns role=customer accounts only. */
export const CustomerLookupPage = () => {
  const { t } = useTranslation();
  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data, isFetching, error } = useCustomerList(query);

  const columns: ColumnsType<User> = [
    { title: t('user.fullName'), dataIndex: 'full_name', key: 'full_name', ellipsis: true },
    { title: t('user.email'), dataIndex: 'email', key: 'email', ellipsis: true },
    {
      title: t('customerLookup.phone'),
      dataIndex: 'phone',
      key: 'phone',
      width: 140,
      render: (value?: string) => value || '-',
    },
    {
      title: t('user.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 100,
      align: 'right',
      render: (_, record) => (
        <Button type="link" onClick={() => setSelectedId(record.id)}>
          {t('customerLookup.viewDetail')}
        </Button>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title={t('customerLookup.title')}
        extra={
          <Input.Search
            allowClear
            defaultValue={search}
            placeholder={t('customerLookup.searchPlaceholder')}
            style={{ width: 320 }}
            onSearch={setSearch}
          />
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
        <Table<User>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          locale={{
            emptyText: search ? (
              t('common.noData')
            ) : (
              <Typography.Text type="secondary">{t('customerLookup.emptyHint')}</Typography.Text>
            ),
          }}
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

      <CustomerDetailDrawer customerId={selectedId} onClose={() => setSelectedId(null)} />
    </>
  );
};

export default CustomerLookupPage;
