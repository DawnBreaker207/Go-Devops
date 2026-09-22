import { useState } from 'react';
import { Alert, Button, DatePicker, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type dayjs from 'dayjs';
import PageHeader from '@/components/PageHeader';
import TableCard from '@/components/TableCard';
import OrderDetailDrawer from './components/OrderDetailDrawer';
import { useAdminOrderList } from './hooks/useBookings';
import { BOOKING_STATUS_COLOR, PAYMENT_STATUS_COLOR } from './constants';
import type { AdminOrder, BookingStatus, PaymentStatus, SoldVia } from '@/types';
import { BOOKING_STATUSES, PAYMENT_STATUSES, SOLD_VIA } from '@/types';
import { useListQuery } from '@/hooks/useListQuery';
import { errorMessage } from '@/utils/error';
import { API_DATE_FORMAT, DATE_FORMAT, formatDateTime, formatVND } from '@/utils/format';

export const BookingsPage = () => {
  const { t } = useTranslation();

  const { query, page, pageSize, search, setPage, setSearch } = useListQuery();
  const [status, setStatus] = useState<BookingStatus>();
  const [paymentStatus, setPaymentStatus] = useState<PaymentStatus>();
  const [soldVia, setSoldVia] = useState<SoldVia>();
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [selected, setSelected] = useState<AdminOrder | null>(null);

  const { data, isFetching, error } = useAdminOrderList({
    ...query,
    status,
    payment_status: paymentStatus,
    sold_via: soldVia,
    from: range?.[0]?.format(API_DATE_FORMAT),
    to: range?.[1]?.format(API_DATE_FORMAT),
  });

  const resetToFirstPage = () => setPage(1);

  const columns: ColumnsType<AdminOrder> = [
    {
      title: t('booking.customer'),
      key: 'customer',
      ellipsis: true,
      render: (_, record) => {
        const c = record.customer;
        if (!c)
          return <Typography.Text type="secondary">{t('booking.walkInNoContact')}</Typography.Text>;
        return (
          <Space direction="vertical" size={0}>
            <span>{c.full_name || c.email || '-'}</span>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {/* Counter sales have no account, hence no email. */}
              {c.email || c.phone || t('booking.walkIn')}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: t('showtime.movie'),
      key: 'movie',
      ellipsis: true,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{record.showtime?.movie_title ?? '-'}</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {record.showtime?.hall_name} · {formatDateTime(record.showtime?.start_at)}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: t('booking.seats'),
      dataIndex: 'seats',
      key: 'seats',
      width: 80,
      align: 'right',
    },
    {
      // payable_amount is what the customer was charged. total_amount is the
      // undiscounted seat subtotal, so showing THAT here would disagree with the
      // payment row on any discounted order.
      title: t('booking.total'),
      dataIndex: 'payable_amount',
      key: 'payable_amount',
      width: 150,
      align: 'right',
      render: (value: number, record) => (
        <Space direction="vertical" size={0} style={{ alignItems: 'flex-end' }}>
          <span className="tabular-nums">{formatVND(value)}</span>
          {record.discount_amount > 0 ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {`\u2212${formatVND(record.discount_amount)}`}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('booking.soldVia'),
      dataIndex: 'sold_via',
      key: 'sold_via',
      width: 110,
      render: (value: SoldVia) => <Tag bordered={false}>{t(`booking.soldVia_${value}`)}</Tag>,
    },
    {
      title: t('booking.status'),
      dataIndex: 'status',
      key: 'status',
      width: 140,
      render: (value: BookingStatus, record) => (
        <Space direction="vertical" size={0}>
          <Tag color={BOOKING_STATUS_COLOR[value]} bordered={false}>
            {t(`booking.status_${value}`)}
          </Tag>
          {record.status_reason ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {t(`booking.reason_${record.status_reason}`, record.status_reason)}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('booking.payment'),
      key: 'payment',
      width: 130,
      // payment is the attempt HOLDING money. Holds without payment are normal, not missing data.
      render: (_, record) =>
        record.payment ? (
          <Tag color={PAYMENT_STATUS_COLOR[record.payment.status]} bordered={false}>
            {t(`booking.payment_${record.payment.status}`)}
          </Tag>
        ) : (
          <Typography.Text type="secondary">{t('booking.noPaymentShort')}</Typography.Text>
        ),
    },
    {
      title: t('booking.createdAt'),
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
        <Button type="link" aria-label={`detail-${record.id}`} onClick={() => setSelected(record)}>
          {t('booking.viewDetail')}
        </Button>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title={t('booking.title')}
        extra={
          <Input.Search
            allowClear
            defaultValue={search}
            placeholder={t('booking.searchPlaceholder')}
            style={{ width: 320 }}
            onSearch={setSearch}
          />
        }
      />

      <Space wrap style={{ marginBottom: 16 }}>
        <Select<BookingStatus>
          allowClear
          style={{ width: 160 }}
          placeholder={t('booking.status')}
          value={status}
          onChange={(value) => {
            setStatus(value);
            resetToFirstPage();
          }}
          options={BOOKING_STATUSES.map((s) => ({ value: s, label: t(`booking.status_${s}`) }))}
        />
        <Select<PaymentStatus>
          allowClear
          style={{ width: 190 }}
          placeholder={t('booking.paymentStatus')}
          value={paymentStatus}
          onChange={(value) => {
            setPaymentStatus(value);
            resetToFirstPage();
          }}
          options={PAYMENT_STATUSES.map((s) => ({ value: s, label: t(`booking.payment_${s}`) }))}
        />
        <Select<SoldVia>
          allowClear
          style={{ width: 150 }}
          placeholder={t('booking.soldVia')}
          value={soldVia}
          onChange={(value) => {
            setSoldVia(value);
            resetToFirstPage();
          }}
          options={SOLD_VIA.map((s) => ({ value: s, label: t(`booking.soldVia_${s}`) }))}
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

      {paymentStatus ? (
        <Alert
          type="info"
          showIcon
          closable
          style={{ marginBottom: 16 }}
          message={t('booking.paymentFilterHint')}
        />
      ) : null}

      {error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(error, t('common.somethingWrong'))}
        />
      ) : null}

      <TableCard>
        <Table<AdminOrder>
          rowKey="id"
          columns={columns}
          dataSource={data?.items ?? []}
          loading={isFetching}
          scroll={{ x: 1200 }}
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

      <OrderDetailDrawer order={selected} onClose={() => setSelected(null)} />
    </>
  );
};

export default BookingsPage;
