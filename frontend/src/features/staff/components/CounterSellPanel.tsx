import { useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  Result,
  Row,
  Select,
  Skeleton,
  Space,
  Typography,
} from 'antd';
import { useTranslation } from 'react-i18next';
import SeatPickerGrid from './SeatPickerGrid';
import OrderDetailView from './OrderDetailView';
import { useCounterSeatMap, useCounterSell, useShowtimeOptions } from '../hooks/useBoxOffice';
import type { OrderDetail, SeatMapSeat } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';

const MAX_SEATS = 8;

interface CounterSellFormValues {
  customer_name?: string;
  customer_phone?: string;
}

/** Counter sale from showtime and seat picks to an immediate order. */
export const CounterSellPanel = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<CounterSellFormValues>();

  const [search, setSearch] = useState('');
  const [showtimeId, setShowtimeId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [sellError, setSellError] = useState<string | null>(null);
  const [sold, setSold] = useState<OrderDetail | null>(null);

  const showtimes = useShowtimeOptions(search || undefined, 'open');
  const seatMap = useCounterSeatMap(showtimeId);
  const counterSell = useCounterSell();

  const selectedSeats = useMemo(
    () =>
      (seatMap.data?.seats ?? []).filter(
        (s) => s.showtime_seat_id && selected.has(s.showtime_seat_id)
      ),
    [seatMap.data, selected]
  );
  const total = selectedSeats.reduce((sum, s) => sum + s.price, 0);

  const toggleSeat = (seat: SeatMapSeat) => {
    const id = seat.showtime_seat_id;
    if (!id) return;
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else if (next.size < MAX_SEATS) next.add(id);
      return next;
    });
  };

  const pickShowtime = (id: string) => {
    setShowtimeId(id);
    setSelected(new Set());
    setSellError(null);
    setSold(null);
  };

  const startOver = () => {
    setSold(null);
    setShowtimeId(null);
    setSelected(new Set());
    form.resetFields();
  };

  const submit = async (values: CounterSellFormValues) => {
    if (!showtimeId || selectedSeats.length === 0) return;
    setSellError(null);
    try {
      const order = await counterSell.mutateAsync({
        show_id: showtimeId,
        seat_ids: selectedSeats.map((s) => s.showtime_seat_id as string),
        customer_name: values.customer_name || undefined,
        customer_phone: values.customer_phone || undefined,
      });
      setSold(order);
      setSelected(new Set());
      message.success(t('boxOffice.sellSuccess'));
    } catch (error) {
      setSellError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  if (sold) {
    return (
      <Card>
        <Result
          status="success"
          title={t('boxOffice.sellSuccess')}
          subTitle={t('boxOffice.sellSuccessHint', { id: sold.id })}
        />
        <OrderDetailView order={sold} />
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Button type="primary" onClick={startOver}>
            {t('boxOffice.sellAnother')}
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <Row gutter={16}>
      <Col xs={24} lg={14}>
        <Card title={t('boxOffice.pickShowtime')} style={{ marginBottom: 16 }}>
          <Select
            showSearch
            allowClear
            style={{ width: '100%' }}
            placeholder={t('boxOffice.pickShowtimePlaceholder')}
            value={showtimeId ?? undefined}
            filterOption={false}
            loading={showtimes.isFetching}
            onSearch={setSearch}
            onChange={(value) => (value ? pickShowtime(value) : setShowtimeId(null))}
            options={(showtimes.data?.items ?? []).map((s) => ({
              value: s.id,
              label: `${s.movie_title} · ${s.hall_name} · ${formatDateTime(s.start_at)}`,
            }))}
          />
        </Card>

        {showtimeId ? (
          <Card title={t('boxOffice.pickSeats')}>
            {seatMap.error ? (
              <Alert
                type="error"
                showIcon
                message={errorMessage(seatMap.error, t('common.somethingWrong'))}
              />
            ) : seatMap.isLoading || !seatMap.data ? (
              <Skeleton active paragraph={{ rows: 4 }} />
            ) : (
              <SeatPickerGrid
                seatMap={seatMap.data}
                selected={selected}
                onToggle={toggleSeat}
                maxSeats={MAX_SEATS}
              />
            )}
          </Card>
        ) : null}
      </Col>

      <Col xs={24} lg={10}>
        <Card title={t('boxOffice.orderSummary')}>
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Typography.Text>
              {t('customer.seatCount', { count: selectedSeats.length, max: MAX_SEATS })}
            </Typography.Text>
            <Typography.Text>
              {selectedSeats.length > 0
                ? selectedSeats.map((s) => s.label).join(', ')
                : t('customer.noSeatPicked')}
            </Typography.Text>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {formatVND(total)}
            </Typography.Title>

            <Form<CounterSellFormValues> form={form} layout="vertical" onFinish={submit}>
              <Form.Item name="customer_name" label={t('boxOffice.customerName')}>
                <Input placeholder={t('boxOffice.customerNamePlaceholder')} maxLength={255} />
              </Form.Item>
              <Form.Item name="customer_phone" label={t('boxOffice.customerPhone')}>
                <Input placeholder={t('boxOffice.customerPhonePlaceholder')} maxLength={20} />
              </Form.Item>

              {sellError ? (
                <Alert type="error" showIcon style={{ marginBottom: 16 }} message={sellError} />
              ) : null}

              <Button
                type="primary"
                htmlType="submit"
                block
                disabled={!showtimeId || selectedSeats.length === 0}
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

export default CounterSellPanel;
