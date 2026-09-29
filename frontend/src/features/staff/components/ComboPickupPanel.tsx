import { useState } from 'react';
import { Alert, App, Button, Card, Empty, Input, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { useCollectComboOrder, usePendingPickups } from '../hooks/useBoxOffice';
import type { ComboPickup } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';

/** Counter handover board: online pre-orders awaiting pickup. */
export const ComboPickupPanel = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const [submitted, setSubmitted] = useState<string | undefined>(undefined);
  const [collectingId, setCollectingId] = useState<string | null>(null);

  const pickups = usePendingPickups(submitted);
  const collect = useCollectComboOrder();

  const doCollect = async (order: ComboPickup) => {
    setCollectingId(order.order_id);
    try {
      await collect.mutateAsync(order.order_id);
      message.success(t('boxOffice.pickupDone', { id: order.order_id.slice(0, 8) }));
    } catch (error) {
      // Lost the race (or already handed over): the list refresh shows the truth.
      message.error(errorMessage(error, t('common.somethingWrong')));
    } finally {
      setCollectingId(null);
    }
  };

  const columns: ColumnsType<ComboPickup> = [
    {
      title: t('boxOffice.pickupCustomer'),
      key: 'customer',
      ellipsis: true,
      render: (_, row) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>
            {row.customer_name || t('booking.walkInNoContact')}
          </Typography.Text>
          {row.customer_email ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {row.customer_email}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('boxOffice.pickupItems'),
      key: 'items',
      render: (_, row) => (
        <Space direction="vertical" size={0}>
          {row.items.map((item) => (
            <span key={item.combo_id} className="tabular-nums">
              {item.combo_name} × {item.quantity}
            </span>
          ))}
          {row.movie_title ? (
            <Tag bordered={false} style={{ marginTop: 4 }}>
              {row.movie_title}
              {row.showtime_at ? ` · ${formatDateTime(row.showtime_at)}` : ''}
            </Tag>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('report.totalSales'),
      dataIndex: 'total',
      key: 'total',
      width: 130,
      align: 'right',
      render: (value: number) => <span className="tabular-nums">{formatVND(value)}</span>,
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 150,
      align: 'right',
      render: (_, row) => (
        <Button
          type="primary"
          loading={collectingId === row.order_id}
          onClick={() => void doCollect(row)}
        >
          {t('boxOffice.pickupCollect')}
        </Button>
      ),
    },
  ];

  return (
    <Card title={t('boxOffice.tabPickup')}>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          style={{ width: 340 }}
          placeholder={t('boxOffice.pickupSearchPlaceholder')}
          onSearch={(value) => setSubmitted(value.trim() || undefined)}
        />
      </Space>

      {pickups.error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(pickups.error, t('common.somethingWrong'))}
        />
      ) : null}

      <Table<ComboPickup>
        rowKey="order_id"
        columns={columns}
        dataSource={pickups.data ?? []}
        loading={pickups.isFetching || collect.isPending}
        pagination={false}
        scroll={{ x: 700 }}
        locale={{ emptyText: <Empty description={t('boxOffice.pickupEmpty')} /> }}
      />
    </Card>
  );
};

export default ComboPickupPanel;
