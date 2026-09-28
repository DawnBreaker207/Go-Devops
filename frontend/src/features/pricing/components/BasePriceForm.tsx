import { Alert, App, Button, Form, InputNumber, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import type { SeatType } from '@/types';
import { SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { formatVND } from '@/utils/format';
import { useBasePrices, useSetBasePrices } from '../hooks/usePricing';

/** Price form scale: whole VND, no minor units. */
const MIN_BASE_PRICE = 1;
const MAX_BASE_PRICE = 100_000_000;

type FormValues = Record<SeatType, number>;

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

/** Inline (not a modal) editor for the 4 global seat-type base prices. Mounted only once the
 *  GET resolves, so `initialValues` is seeded from real data - there is no `destroyOnHidden`
 *  remount to lean on here the way a modal has (see pages-components.md's documented bug about
 *  seeding via a useEffect instead). */
const BasePriceFields = ({ initialValues }: { initialValues: Partial<FormValues> }) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const setBasePrices = useSetBasePrices();

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await setBasePrices.mutateAsync({ prices: values });
      message.success(t('pricing.baseSaved'));
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  return (
    <Form<FormValues> form={form} layout="vertical" initialValues={initialValues}>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
        {SEAT_TYPES.map((type) => (
          <Form.Item
            key={type}
            name={type}
            label={t(`hall.seatType_${type}`)}
            rules={[{ required: true, message: t('common.requiredField') }]}
            style={{ flex: '1 1 150px', marginBottom: 8 }}
          >
            <InputNumber<number>
              min={MIN_BASE_PRICE}
              max={MAX_BASE_PRICE}
              step={5000}
              style={{ width: '100%' }}
              formatter={(value) => (value ? formatVND(Number(value)) : '')}
              parser={(value) => Number((value ?? '').replace(/\D/g, '')) || 0}
            />
          </Form.Item>
        ))}
      </div>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" loading={setBasePrices.isPending} onClick={() => void handleSave()}>
          {t('common.save')}
        </Button>
      </Form.Item>
    </Form>
  );
};

export const BasePriceForm = () => {
  const { t } = useTranslation();
  const basePrices = useBasePrices();

  if (basePrices.isLoading) return <Spin />;

  // Display order follows SEAT_TYPES, not the server's return order (alphabetical).
  const initialValues: Partial<FormValues> = SEAT_TYPES.reduce<Partial<FormValues>>((acc, type) => {
    const row = basePrices.data?.find((p) => p.seat_type === type);
    if (row) acc[type] = row.price;
    return acc;
  }, {});

  return (
    <div style={{ maxWidth: 720, marginBottom: 24 }}>
      {basePrices.error ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={errorMessage(basePrices.error, t('common.somethingWrong'))}
        />
      ) : null}
      <BasePriceFields initialValues={initialValues} />
    </div>
  );
};

export default BasePriceForm;
