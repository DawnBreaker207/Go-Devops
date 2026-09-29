import { useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Input,
  List,
  Modal,
  Skeleton,
  Space,
  Tag,
  Typography,
} from 'antd';
import { useTranslation } from 'react-i18next';
import OrderDetailView from './OrderDetailView';
import PrintTicketsButton from './PrintTicketsButton';
import QrScanner from './QrScanner';
import {
  useCollectOrderTickets,
  useStaffCustomerOrders,
  useStaffCustomerSearch,
  useStaffOrderByTicket,
  useStaffOrderLookup,
} from '../hooks/useBoxOffice';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';

/** Counter reception: find an order by id, ticket scan or customer phone, print it, mark it handed over. */
export const OrderLookupPanel = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [id, setId] = useState('');
  const [scanning, setScanning] = useState(false);
  const [phone, setPhone] = useState('');
  const [submittedPhone, setSubmittedPhone] = useState<string | undefined>(undefined);
  const [customerId, setCustomerId] = useState<string | undefined>(undefined);
  const lookup = useStaffOrderLookup();
  const lookupByTicket = useStaffOrderByTicket();
  const collect = useCollectOrderTickets();
  const customers = useStaffCustomerSearch(submittedPhone);
  const customerOrders = useStaffCustomerOrders(customerId);

  const search = () => {
    const trimmed = id.trim();
    if (!trimmed) return;
    lookup.mutate(trimmed);
  };

  const searchScanned = (code: string) => {
    setScanning(false);
    setId(code);
    lookupByTicket.mutate(code);
  };

  const searchPhone = () => {
    const trimmed = phone.trim();
    if (!trimmed) return;
    setCustomerId(undefined);
    setSubmittedPhone(trimmed);
  };

  const pickCustomer = (cid: string) => {
    setCustomerId(cid);
  };

  const pickOrder = (orderId: string) => {
    setId(orderId);
    lookup.mutate(orderId);
  };

  const markCollected = async (orderId: string) => {
    try {
      await collect.mutateAsync(orderId);
      // Refresh the shown detail in place so the stamp appears at once.
      lookup.mutate(orderId);
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const active = lookupByTicket.isPending ? lookupByTicket : lookup;
  const shown = active.isSuccess ? active.data : undefined;

  return (
    <Card title={t('boxOffice.lookupOrder')}>
      <Space direction="vertical" size="middle" style={{ width: '100%', marginBottom: 16 }}>
        <div style={{ maxWidth: 480 }}>
          <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {t('boxOffice.lookupByOrder')}
          </Typography.Text>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              placeholder={t('boxOffice.lookupOrderPlaceholder')}
              value={id}
              onChange={(e) => setId(e.target.value)}
              onPressEnter={search}
            />
            <Button type="primary" loading={lookup.isPending} onClick={search}>
              {t('common.search')}
            </Button>
            <Button onClick={() => setScanning(true)}>{t('boxOffice.scanQr')}</Button>
          </Space.Compact>
        </div>
        <div style={{ maxWidth: 480 }}>
          <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {t('boxOffice.lookupByPhone')}
          </Typography.Text>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              placeholder={t('boxOffice.lookupPhonePlaceholder')}
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              onPressEnter={searchPhone}
            />
            <Button loading={customers.isFetching} onClick={searchPhone}>
              {t('boxOffice.lookupPhone')}
            </Button>
          </Space.Compact>
        </div>
      </Space>
      <Modal
        open={scanning}
        onCancel={() => setScanning(false)}
        title={t('boxOffice.scanQr')}
        footer={null}
        destroyOnHidden
      >
        {scanning ? <QrScanner onScan={searchScanned} /> : null}
      </Modal>

      {customers.data && submittedPhone ? (
        <List
          size="small"
          style={{ maxWidth: 480, marginBottom: 16 }}
          loading={customers.isFetching}
          dataSource={customers.data.items}
          locale={{ emptyText: t('common.noData') }}
          renderItem={(customer) => (
            <List.Item
              actions={[
                <Button
                  key="pick"
                  size="small"
                  type={customerId === customer.id ? 'primary' : 'default'}
                  onClick={() => pickCustomer(customer.id)}
                >
                  {t('boxOffice.lookupCustomerOrders')}
                </Button>,
              ]}
            >
              <Space direction="vertical" size={0}>
                <Typography.Text strong>{customer.full_name}</Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {customer.email}
                  {customer.phone ? ` · ${customer.phone}` : ''}
                </Typography.Text>
              </Space>
            </List.Item>
          )}
        />
      ) : null}

      {customerId ? (
        <List
          size="small"
          style={{ maxWidth: 480, marginBottom: 16 }}
          loading={customerOrders.isFetching}
          dataSource={customerOrders.data?.items ?? []}
          locale={{ emptyText: t('common.noData') }}
          renderItem={(order) => (
            <List.Item
              actions={[
                <Button key="open" size="small" onClick={() => pickOrder(order.id)}>
                  {t('booking.viewDetail')}
                </Button>,
              ]}
            >
              <Space size={6}>
                <Typography.Text className="tabular-nums">
                  {formatVND(order.payable_amount)}
                </Typography.Text>
                <Tag bordered={false}>{t(`booking.status_${order.status}`, order.status)}</Tag>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {formatDateTime(order.created_at)}
                </Typography.Text>
              </Space>
            </List.Item>
          )}
        />
      ) : null}

      {active.isPending ? <Skeleton active paragraph={{ rows: 4 }} /> : null}

      {active.isError ? (
        <Alert
          type="error"
          showIcon
          message={errorMessage(active.error, t('common.somethingWrong'))}
        />
      ) : null}

      {active.isSuccess && shown ? (
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          {shown.collected_at ? (
            <Alert
              type="success"
              showIcon
              message={t('boxOffice.collectedAt', { time: formatDateTime(shown.collected_at) })}
            />
          ) : (
            <div>
              <Button
                type="primary"
                loading={collect.isPending}
                onClick={() => void markCollected(shown.id)}
              >
                {t('boxOffice.collectTickets')}
              </Button>
            </div>
          )}
          <OrderDetailView order={shown} />
          <div>
            <PrintTicketsButton order={shown} />
          </div>
        </Space>
      ) : null}
    </Card>
  );
};

export default OrderLookupPanel;
