import { Alert, Descriptions, Drawer, Skeleton, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { useCustomerOrders, useCustomerProfile } from '../hooks/useCustomerLookup';
import { useListQuery } from '@/hooks/useListQuery';
import type { BookingStatus, OrderStatus } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';

const BOOKING_STATUS_COLOR: Record<BookingStatus, string> = {
  pending: 'gold',
  confirmed: 'green',
  expired: 'default',
  refunded: 'red',
};

interface CustomerDetailDrawerProps {
  /** Id of the customer being viewed; null means the drawer is closed. */
  customerId: string | null;
  onClose: () => void;
}

/** GET /staff/customers/:id + GET /staff/customers/:id/orders in one drawer. */
export const CustomerDetailDrawer = ({ customerId, onClose }: CustomerDetailDrawerProps) => {
  const { t } = useTranslation();
  const { query, page, pageSize, setPage } = useListQuery();

  const profile = useCustomerProfile(customerId);
  const orders = useCustomerOrders(customerId, query);

  const columns: ColumnsType<OrderStatus> = [
    {
      title: t('booking.code'),
      dataIndex: 'id',
      key: 'id',
      ellipsis: true,
      render: (value: string) => <Typography.Text copyable>{value}</Typography.Text>,
    },
    {
      title: t('showtime.movie'),
      key: 'movie',
      render: (_, record) => record.showtime?.movie_title ?? '-',
    },
    {
      title: t('booking.status'),
      dataIndex: 'status',
      key: 'status',
      width: 140,
      render: (value: BookingStatus) => (
        <Tag color={BOOKING_STATUS_COLOR[value]} bordered={false}>
          {t(`booking.status_${value}`)}
        </Tag>
      ),
    },
    {
      title: t('booking.total'),
      // What the customer was charged, not the undiscounted seat subtotal.
      dataIndex: 'payable_amount',
      key: 'payable_amount',
      width: 120,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
    {
      title: t('booking.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (value: string) => formatDateTime(value),
    },
  ];

  return (
    <Drawer
      open={Boolean(customerId)}
      onClose={onClose}
      width={640}
      destroyOnHidden
      title={t('customerLookup.detailTitle')}
    >
      {profile.error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(profile.error, t('common.somethingWrong'))}
        />
      ) : null}

      {profile.isLoading || !profile.data ? (
        <Skeleton active paragraph={{ rows: 4 }} />
      ) : (
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('user.fullName')}>
              {profile.data.full_name}
            </Descriptions.Item>
            <Descriptions.Item label={t('user.email')}>{profile.data.email}</Descriptions.Item>
            <Descriptions.Item label={t('customerLookup.phone')}>
              {profile.data.phone || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('user.status')}>
              <Tag color={profile.data.active ? 'green' : 'default'} bordered={false}>
                {profile.data.active ? t('user.statusActive') : t('user.statusLocked')}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('user.createdAt')}>
              {formatDateTime(profile.data.created_at)}
            </Descriptions.Item>
          </Descriptions>

          {orders.error ? (
            <Alert
              type="error"
              showIcon
              message={errorMessage(orders.error, t('common.somethingWrong'))}
            />
          ) : (
            <Table<OrderStatus>
              rowKey="id"
              size="small"
              title={() => t('customerLookup.orderHistory')}
              columns={columns}
              dataSource={orders.data?.items ?? []}
              loading={orders.isFetching}
              pagination={{
                current: orders.data?.meta.page ?? page,
                pageSize: orders.data?.meta.page_size ?? pageSize,
                total: orders.data?.meta.total ?? 0,
                onChange: setPage,
              }}
            />
          )}
        </Space>
      )}
    </Drawer>
  );
};

export default CustomerDetailDrawer;
