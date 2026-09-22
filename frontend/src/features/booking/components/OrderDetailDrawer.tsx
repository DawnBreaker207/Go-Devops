import { Alert, Descriptions, Drawer, Skeleton, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type { AdminOrder, Ticket } from '@/types';
import { useOrderDetail } from '../hooks/useBookings';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';
import { BOOKING_STATUS_COLOR, PAYMENT_STATUS_COLOR } from '../constants';

interface OrderDetailDrawerProps {
  /** Table row that opened the drawer; null means the drawer is closed. */
  order: AdminOrder | null;
  onClose: () => void;
}

/** Order detail drawer combining live detail with the opening row. */
export const OrderDetailDrawer = ({ order, onClose }: OrderDetailDrawerProps) => {
  const { t } = useTranslation();
  const { data, isFetching, error } = useOrderDetail(order?.id ?? null);

  const customer = order?.customer;
  const detail = data ?? order ?? undefined;

  const ticketColumns: ColumnsType<Ticket> = [
    { title: t('booking.seat'), dataIndex: 'seat_label', key: 'seat_label', width: 80 },
    {
      title: t('booking.seatType'),
      dataIndex: 'seat_type',
      key: 'seat_type',
      width: 110,
      render: (value: string) => t(`booking.seatType_${value}`, value),
    },
    {
      title: t('booking.price'),
      dataIndex: 'price',
      key: 'price',
      width: 120,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
    {
      title: t('booking.ticketStatus'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value: string) => (
        <Tag color={value === 'redeemed' ? 'blue' : 'green'} bordered={false}>
          {t(`booking.ticket_${value}`, value)}
        </Tag>
      ),
    },
  ];

  return (
    <Drawer
      open={Boolean(order)}
      onClose={onClose}
      width={620}
      destroyOnHidden
      title={t('booking.detailTitle')}
    >
      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      {detail ? (
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('booking.code')}>
              <Typography.Text copyable>{detail.id}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('booking.status')}>
              <Space size={6}>
                <Tag color={BOOKING_STATUS_COLOR[detail.status]} bordered={false}>
                  {t(`booking.status_${detail.status}`)}
                </Tag>
                {detail.status_reason ? (
                  <Typography.Text type="secondary">
                    {t(`booking.reason_${detail.status_reason}`, detail.status_reason)}
                  </Typography.Text>
                ) : null}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label={t('booking.soldVia')}>
              {order ? t(`booking.soldVia_${order.sold_via}`) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('booking.customer')}>
              {customer ? (
                <Space direction="vertical" size={0}>
                  <span>{customer.full_name || customer.email || '-'}</span>
                  {customer.email && customer.full_name ? (
                    <Typography.Text type="secondary">{customer.email}</Typography.Text>
                  ) : null}
                  {customer.phone ? (
                    <Typography.Text type="secondary">{customer.phone}</Typography.Text>
                  ) : null}
                  {/* Counter sales have no account; say so instead of leaving it blank. */}
                  {!customer.user_id ? (
                    <Typography.Text type="secondary">{t('booking.walkIn')}</Typography.Text>
                  ) : null}
                </Space>
              ) : (
                <Typography.Text type="secondary">{t('booking.walkInNoContact')}</Typography.Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label={t('showtime.movie')}>
              {detail.showtime?.movie_title ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('showtime.hall')}>
              {detail.showtime?.hall_name ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('showtime.startAt')}>
              {formatDateTime(detail.showtime?.start_at)}
            </Descriptions.Item>
            <Descriptions.Item label={t('booking.createdAt')}>
              {formatDateTime(detail.created_at)}
            </Descriptions.Item>
            {/* Three separate facts. The discount row only appears when there
                is one, so an ordinary order reads exactly as it did before. */}
            <Descriptions.Item label={t('booking.subtotal')}>
              <span className="tabular-nums">{formatVND(detail.total_amount)}</span>
            </Descriptions.Item>
            {detail.discount_amount > 0 ? (
              <Descriptions.Item label={t('booking.discount')}>
                <span className="tabular-nums">{`\u2212${formatVND(detail.discount_amount)}`}</span>
              </Descriptions.Item>
            ) : null}
            <Descriptions.Item label={t('booking.total')}>
              <span className="tabular-nums">{formatVND(detail.payable_amount)}</span>
            </Descriptions.Item>
          </Descriptions>

          {detail.payment ? (
            <Descriptions column={1} size="small" bordered title={t('booking.payment')}>
              <Descriptions.Item label={t('booking.provider')}>
                {detail.payment.provider}
              </Descriptions.Item>
              <Descriptions.Item label={t('booking.txnRef')}>
                <Typography.Text copyable>{detail.payment.txn_ref}</Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('booking.paymentStatus')}>
                <Tag color={PAYMENT_STATUS_COLOR[detail.payment.status]} bordered={false}>
                  {t(`booking.payment_${detail.payment.status}`)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('booking.paidAt')}>
                {formatDateTime(detail.payment.paid_at)}
              </Descriptions.Item>
            </Descriptions>
          ) : (
            <Alert type="info" showIcon message={t('booking.noPayment')} />
          )}

          {isFetching && !data ? (
            <Skeleton active paragraph={{ rows: 3 }} />
          ) : (
            <Table<Ticket>
              rowKey="id"
              size="small"
              title={() => t('booking.tickets')}
              columns={ticketColumns}
              dataSource={data?.tickets ?? []}
              pagination={false}
            />
          )}
        </Space>
      ) : (
        <Skeleton active paragraph={{ rows: 6 }} />
      )}
    </Drawer>
  );
};

export default OrderDetailDrawer;
