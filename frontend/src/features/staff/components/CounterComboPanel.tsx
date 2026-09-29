import { useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  Radio,
  Result,
  Row,
  Space,
  Table,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { useCombos } from '@/features/booking-flow/hooks/useCombos';
import { useCounterComboSell } from '../hooks/useBoxOffice';
import type { Combo, ComboPayMethod } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';

const MAX_QTY = 20;

interface CounterComboFormValues {
  customer_name?: string;
  pay_method: ComboPayMethod;
}

/** Walk-in concession sale at the counter (no ticket, no account). */
export const CounterComboPanel = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<CounterComboFormValues>();

  const [qty, setQty] = useState<Record<string, number>>({});
  const [sellError, setSellError] = useState<string | null>(null);
  const [soldId, setSoldId] = useState<string | null>(null);
  const [soldTotal, setSoldTotal] = useState(0);

  const combos = useCombos();
  const counterSell = useCounterComboSell();

  const lines = useMemo(
    () =>
      (combos.data ?? [])
        .map((c) => ({ combo: c, quantity: qty[c.id] ?? 0 }))
        .filter((l) => l.quantity > 0),
    [combos, qty]
  );
  const total = lines.reduce((sum, l) => sum + l.combo.price * l.quantity, 0);

  const changeQty = (comboId: string, delta: number) => {
    setQty((current) => {
      const next = Math.min(MAX_QTY, Math.max(0, (current[comboId] ?? 0) + delta));
      return { ...current, [comboId]: next };
    });
  };

  const submit = async (values: CounterComboFormValues) => {
    if (lines.length === 0) return;
    setSellError(null);
    try {
      const order = await counterSell.mutateAsync({
        items: lines.map((l) => ({ combo_id: l.combo.id, quantity: l.quantity })),
        pay_method: values.pay_method,
        customer_name: values.customer_name || undefined,
      });
      setSoldId(order.id);
      setSoldTotal(order.total);
      setQty({});
      form.resetFields();
      message.success(t('boxOffice.comboSold'));
    } catch (error) {
      setSellError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const startOver = () => {
    setSoldId(null);
    setSoldTotal(0);
  };

  if (soldId) {
    return (
      <Card>
        <Result
          status="success"
          title={t('boxOffice.comboSold')}
          subTitle={t('boxOffice.sellSuccessHint', { id: soldId })}
        />
        <div style={{ textAlign: 'center' }}>
          <Typography.Title level={4} style={{ margin: 0 }}>
            {formatVND(soldTotal)}
          </Typography.Title>
        </div>
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Button type="primary" onClick={startOver}>
            {t('boxOffice.sellAnother')}
          </Button>
        </div>
      </Card>
    );
  }

  const columns: ColumnsType<{ combo: Combo; quantity: number }> = [
    { title: t('concession.name'), dataIndex: ['combo', 'name'], key: 'name', ellipsis: true },
    {
      title: t('concession.price'),
      key: 'price',
      width: 130,
      align: 'right',
      render: (_, row) => <span className="tabular-nums">{formatVND(row.combo.price)}</span>,
    },
    {
      title: t('boxOffice.comboQty'),
      key: 'qty',
      width: 170,
      align: 'center',
      render: (_, row) => (
        <Space>
          <Button
            size="small"
            onClick={() => changeQty(row.combo.id, -1)}
            disabled={(qty[row.combo.id] ?? 0) === 0}
          >
            −
          </Button>
          <span className="tabular-nums" style={{ minWidth: 24, textAlign: 'center' }}>
            {qty[row.combo.id] ?? 0}
          </span>
          <Button
            size="small"
            onClick={() => changeQty(row.combo.id, 1)}
            disabled={(qty[row.combo.id] ?? 0) >= MAX_QTY}
          >
            +
          </Button>
        </Space>
      ),
    },
    {
      title: t('report.totalSales'),
      key: 'subtotal',
      width: 150,
      align: 'right',
      render: (_, row) => (
        <span className="tabular-nums">{formatVND(row.combo.price * row.quantity)}</span>
      ),
    },
  ];

  return (
    <Row gutter={16}>
      <Col xs={24} lg={14}>
        <Card title={t('boxOffice.comboPick')}>
          {combos.error ? (
            <Alert
              type="error"
              showIcon
              style={{ marginBottom: 16 }}
              message={errorMessage(combos.error, t('common.somethingWrong'))}
            />
          ) : null}
          <Table<{ combo: Combo; quantity: number }>
            rowKey={(row) => row.combo.id}
            columns={columns}
            dataSource={(combos.data ?? []).map((combo) => ({
              combo,
              quantity: qty[combo.id] ?? 0,
            }))}
            loading={combos.isLoading}
            pagination={false}
            locale={{ emptyText: <Empty description={t('common.noData')} /> }}
          />
        </Card>
      </Col>

      <Col xs={24} lg={10}>
        <Card title={t('boxOffice.orderSummary')}>
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Typography.Text>
              {lines.length > 0
                ? lines.map((l) => `${l.combo.name} × ${l.quantity}`).join(', ')
                : t('boxOffice.comboEmpty')}
            </Typography.Text>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {formatVND(total)}
            </Typography.Title>

            <Form<CounterComboFormValues>
              form={form}
              layout="vertical"
              onFinish={submit}
              initialValues={{ pay_method: 'cash' as ComboPayMethod }}
            >
              <Form.Item name="customer_name" label={t('boxOffice.customerName')}>
                <Input placeholder={t('boxOffice.customerNamePlaceholder')} maxLength={255} />
              </Form.Item>
              <Form.Item
                name="pay_method"
                label={t('boxOffice.payMethod')}
                rules={[{ required: true, message: t('common.requiredField') }]}
              >
                <Radio.Group>
                  <Radio value="cash">{t('boxOffice.payCash')}</Radio>
                  <Radio value="pos">{t('boxOffice.payPos')}</Radio>
                </Radio.Group>
              </Form.Item>

              {sellError ? (
                <Alert type="error" showIcon style={{ marginBottom: 16 }} message={sellError} />
              ) : null}

              <Button
                type="primary"
                htmlType="submit"
                block
                disabled={lines.length === 0}
                loading={counterSell.isPending}
              >
                {t('boxOffice.confirmSell')}
              </Button>
            </Form>
          </Space>
        </Card>
      </Col>
    </Row>
  );
};

export default CounterComboPanel;
