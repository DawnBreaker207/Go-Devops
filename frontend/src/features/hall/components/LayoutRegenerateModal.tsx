import { App, Alert, Form, Input, InputNumber, Modal, Select, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type {
  Hall,
  HallPayload,
  HallPrice,
  HallTemplateName,
  ScreenPosition,
  SeatType,
} from '@/types';
import { SCREEN_POSITIONS, SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { useHallTemplates, useRegenerateLayout } from '../hooks/useHalls';
import { MAX_ROWS, MAX_SEATS_PER_ROW } from '../constants';

interface FormValues {
  template?: HallTemplateName | '';
  rows?: number;
  seats_per_row?: number;
  screen_position: ScreenPosition;
  aisle_after_cols: string;
}

interface LayoutRegenerateModalProps {
  open: boolean;
  hall: Hall;
  /** Current prices, echoed back into the body - see the note in handleOk. */
  prices: HallPrice[];
  onCancel: () => void;
  onDone: () => void;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

/** "4, 10" -> [4, 10]. Drops anything that isn't a positive number. */
const parseAisles = (raw: string): number[] =>
  raw
    .split(',')
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((value) => Number.isInteger(value) && value > 0);

export const LayoutRegenerateModal = ({
  open,
  hall,
  prices,
  onCancel,
  onDone,
}: LayoutRegenerateModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const templates = useHallTemplates();
  const regenerate = useRegenerateLayout();

  const template = Form.useWatch('template', form);
  const usingTemplate = Boolean(template);

  // Seed via initialValues, never setFieldsValue in an effect (StrictMode +
  // preserve={false} wipes values). screen_position and aisle_after_cols MUST
  // be pre-filled: empty resets to 'front' and clears all aisles silently.
  const initialValues: FormValues = {
    template: '',
    rows: hall.rows,
    seats_per_row: hall.seats_per_row,
    screen_position: hall.screen_position,
    aisle_after_cols: hall.aisle_after_cols.join(', '),
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();

      /** Resend name and full prices alongside the layout change. */
      const priceMap = SEAT_TYPES.reduce<Record<SeatType, number>>(
        (acc, type) => {
          acc[type] = prices.find((p) => p.seat_type === type)?.price ?? 1;
          return acc;
        },
        {} as Record<SeatType, number>
      );

      /** With a template, omit layout fields so template values apply. */
      const payload: HallPayload = values.template
        ? { name: hall.name, prices: priceMap, template: values.template }
        : {
            name: hall.name,
            prices: priceMap,
            rows: values.rows,
            seats_per_row: values.seats_per_row,
            // Without a template both fields MUST be sent: RegenerateLayout
            // assigns `current.ScreenPosition = req.ScreenPosition` directly; omitting
            // them resets to 'front' and clears all aisles.
            screen_position: values.screen_position,
            aisle_after_cols: parseAisles(values.aisle_after_cols),
          };

      await regenerate.mutateAsync({ id: hall.id, payload });
      message.success(t('hall.regenerateSuccess'));
      onDone();
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  return (
    <Modal
      open={open}
      title={t('hall.regenerateTitle')}
      okText={t('hall.regenerateConfirm')}
      okButtonProps={{ danger: true }}
      cancelText={t('common.cancel')}
      confirmLoading={regenerate.isPending}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={560}
    >
      {/* Warn upfront instead of after a failed save. */}
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        message={t('hall.regenerateWarnTitle')}
        description={t('hall.regenerateWarnBody')}
      />

      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        <Form.Item name="template" label={t('hall.template')} extra={t('hall.templateHint')}>
          <Select
            loading={templates.isFetching}
            options={[
              { value: '', label: t('hall.templateNone') },
              ...(templates.data ?? []).map((tpl) => ({
                value: tpl.name,
                label: `${t(`hall.template_${tpl.name}`)} — ${tpl.rows}×${tpl.seats_per_row}, ${t(
                  'hall.templateSeats',
                  { count: tpl.seat_count }
                )}`,
              })),
            ]}
          />
        </Form.Item>

        {/* With a template, rows and seats come from the template. */}
        {usingTemplate ? (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message={t('hall.templateOverrides')}
          />
        ) : (
          <>
            <Form.Item
              name="rows"
              label={t('hall.rows')}
              rules={[{ required: true, message: t('common.requiredField') }]}
            >
              <InputNumber min={1} max={MAX_ROWS} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item
              name="seats_per_row"
              label={t('hall.seatsPerRow')}
              rules={[{ required: true, message: t('common.requiredField') }]}
            >
              <InputNumber min={1} max={MAX_SEATS_PER_ROW} style={{ width: '100%' }} />
            </Form.Item>
          </>
        )}

        {/* Hidden once a template is picked: these fields are no longer sent, so showing
            them with stale values would mislead the user. */}
        {usingTemplate ? null : (
          <>
            <Form.Item name="screen_position" label={t('hall.screenPosition')}>
              <Select
                options={SCREEN_POSITIONS.map((value) => ({
                  value,
                  label: t(`hall.screen_${value}`),
                }))}
              />
            </Form.Item>

            <Form.Item
              name="aisle_after_cols"
              label={t('hall.aisles')}
              extra={t('hall.aislesHint')}
            >
              <Input placeholder="4, 10" />
            </Form.Item>
          </>
        )}
      </Form>

      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {t('hall.regenerateScopeNote')}
      </Typography.Text>
    </Modal>
  );
};

export default LayoutRegenerateModal;
