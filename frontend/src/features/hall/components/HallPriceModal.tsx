import { Alert, App, Form, InputNumber, Modal, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import type { Hall, SeatType } from '@/types';
import { SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { formatVND } from '@/utils/format';
import { useHallPrices, useSetHallPrices } from '../hooks/useHalls';
import { MAX_SEAT_PRICE, MIN_SEAT_PRICE } from '../constants';

interface FormValues {
  prices: Record<SeatType, number>;
}

interface HallPriceModalProps {
  open: boolean;
  hall: Hall | null;
  onCancel: () => void;
  onDone: () => void;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

/** Price editor always saving the full set of seat prices. */
export const HallPriceModal = ({ open, hall, onCancel, onDone }: HallPriceModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const prices = useHallPrices(open && hall ? hall.id : undefined);
  const setPrices = useSetHallPrices();

  // Display order follows SEAT_TYPES, not the server's return order:
  // GET sorts alphabetically, PUT echoes back in yet another order; trusting either drifts.
  const initialValues: Partial<FormValues> = prices.data
    ? {
        prices: SEAT_TYPES.reduce<Record<SeatType, number>>(
          (acc, type) => {
            const row = prices.data.find((p) => p.seat_type === type);
            if (row) acc[type] = row.price;
            return acc;
          },
          {} as Record<SeatType, number>
        ),
      }
    : {};

  const missing = prices.data
    ? SEAT_TYPES.filter((type) => {
        const row = prices.data.find((p) => p.seat_type === type);
        return row === undefined || row.price <= 0;
      })
    : [];

  const handleOk = async () => {
    if (!hall) return;
    try {
      const values = await form.validateFields();
      await setPrices.mutateAsync({ id: hall.id, payload: { prices: values.prices } });
      message.success(t('hall.pricesSaved'));
      onDone();
    } catch (error) {
      if (isFormValidationError(error)) return;
      // A missing price type comes back as details {seat_type: "<type>"}, not
      // a form field name, so applyApiFieldErrors can't attach it anywhere -
      // falling through to a toast is correct.
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  return (
    <Modal
      open={open}
      title={hall ? t('hall.pricesTitle', { name: hall.name }) : t('hall.prices')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={setPrices.isPending}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={480}
    >
      {prices.isLoading ? (
        <Spin />
      ) : (
        <>
          {missing.length > 0 ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message={t('hall.pricesIncompleteTitle')}
              description={t('hall.pricesIncompleteBody', {
                types: missing.map((type) => t(`hall.seatType_${type}`)).join(', '),
              })}
            />
          ) : null}

          <Form<FormValues>
            form={form}
            layout="vertical"
            preserve={false}
            initialValues={initialValues}
          >
            {SEAT_TYPES.map((type) => (
              <Form.Item
                key={type}
                name={['prices', type]}
                label={t(`hall.seatType_${type}`)}
                rules={[{ required: true, message: t('common.requiredField') }]}
              >
                <InputNumber<number>
                  min={MIN_SEAT_PRICE}
                  max={MAX_SEAT_PRICE}
                  step={5000}
                  style={{ width: '100%' }}
                  formatter={(value) => (value ? formatVND(Number(value)) : '')}
                  parser={(value) => Number((value ?? '').replace(/\D/g, '')) || 0}
                />
              </Form.Item>
            ))}
          </Form>
        </>
      )}
    </Modal>
  );
};

export default HallPriceModal;
