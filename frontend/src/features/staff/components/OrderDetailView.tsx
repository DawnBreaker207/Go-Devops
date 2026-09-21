import { Descriptions, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type { BookingStatus, OrderDetail, PaymentStatus, Ticket } from '@/types';
import { formatDateTime, formatVND } from '@/utils/format';

const BOOKING_STATUS_COLOR: Record<BookingStatus, string> = {
  pending: 'gold',
  confirmed: 'green',
  expired: 'default',
  refunded: 'red',
};

const PAYMENT_STATUS_COLOR: Record<PaymentStatus, string> = {
  pending: 'gold',
  paid: 'green',
  failed: 'red',
  refund_pending: 'orange',
  refunded: 'purple',
};

interface OrderDetailViewProps {
  order: OrderDetail;
}

/** Read-only view of one sold order with its tickets. */
export const OrderDetailView = ({ order }: OrderDetailViewProps) => {
  const { t } = useTranslation();

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
      width: 110,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
    {
      title: t('boxOffice.ticketCode'),
      dataIndex: 'code',
      key: 'code',
      render: (value: string) => <Typography.Text copyable>{value}</Typography.Text>,
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
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label={t('booking.code')}>
          <Typography.Text copyable>{order.id}</Typography.Text>
        </Descriptions.Item>
        <Descriptions.Item label={t('booking.status')}>
          <Space size={6}>
            <Tag color={BOOKING_STATUS_COLOR[order.status]} bordered={false}>
              {t(`booking.status_${order.status}`)}
            </Tag>
            {order.status_reason ? (
              <Typography.Text type="secondary">
                {t(`booking.reason_${order.status_reason}`, order.status_reason)}
              </Typography.Text>
            ) : null}
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label={t('showtime.movie')}>
          {order.showtime?.movie_title ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('showtime.hall')}>
          {order.showtime?.hall_name ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('showtime.startAt')}>
          {formatDateTime(order.showtime?.start_at)}
        </Descriptions.Item>
        <Descriptions.Item label={t('booking.createdAt')}>
          {formatDateTime(order.created_at)}
        </Descriptions.Item>
        <Descriptions.Item label={t('booking.total')}>
          <span className="tabular-nums">{formatVND(order.total_amount)}</span>
        </Descriptions.Item>
      </Descriptions>

      {order.payment ? (
        <Descriptions column={1} size="small" bordered title={t('booking.payment')}>
          <Descriptions.Item label={t('booking.provider')}>
            {order.payment.provider}
          </Descriptions.Item>
          <Descriptions.Item label={t('booking.paymentStatus')}>
            <Tag color={PAYMENT_STATUS_COLOR[order.payment.status]} bordered={false}>
              {t(`booking.payment_${order.payment.status}`)}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('booking.paidAt')}>
            {formatDateTime(order.payment.paid_at)}
          </Descriptions.Item>
        </Descriptions>
      ) : null}

      <Table<Ticket>
        rowKey="id"
        size="small"
        title={() => t('booking.tickets')}
        columns={ticketColumns}
        dataSource={order.tickets}
        pagination={false}
      />
    </Space>
  );
};

export default OrderDetailView;
