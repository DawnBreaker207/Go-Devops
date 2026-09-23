import { useState } from 'react';
import { Alert, Button, Card, Input, Skeleton, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import OrderDetailView from './OrderDetailView';
import { useStaffOrderLookup } from '../hooks/useBoxOffice';
import { errorMessage } from '@/utils/error';

/** Looks up any order by id: GET /staff/orders/:id. */
export const OrderLookupPanel = () => {
  const { t } = useTranslation();
  const [id, setId] = useState('');
  const lookup = useStaffOrderLookup();

  const search = () => {
    const trimmed = id.trim();
    if (!trimmed) return;
    lookup.mutate(trimmed);
  };

  return (
    <Card title={t('boxOffice.lookupOrder')}>
      <Space.Compact style={{ width: '100%', maxWidth: 480, marginBottom: 16 }}>
        <Input
          placeholder={t('boxOffice.lookupOrderPlaceholder')}
          value={id}
          onChange={(e) => setId(e.target.value)}
          onPressEnter={search}
        />
        <Button type="primary" loading={lookup.isPending} onClick={search}>
          {t('common.search')}
        </Button>
      </Space.Compact>

      {lookup.isPending ? <Skeleton active paragraph={{ rows: 4 }} /> : null}

      {lookup.isError ? (
        <Alert
          type="error"
          showIcon
          message={errorMessage(lookup.error, t('common.somethingWrong'))}
        />
      ) : null}

      {lookup.isSuccess && lookup.data ? <OrderDetailView order={lookup.data} /> : null}
    </Card>
  );
};

export default OrderLookupPanel;
