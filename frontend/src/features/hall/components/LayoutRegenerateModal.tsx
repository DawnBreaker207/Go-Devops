import { App, Alert, Form, Input, InputNumber, Modal, Select, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type { Hall, HallPayload, HallTemplateName, ScreenPosition } from '@/types';
import { SCREEN_POSITIONS } from '@/types';
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
  onCancel: () => void;
  onDone: () => void;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

const parseAisles = (raw: string): number[] =>
  raw
    .split(',')
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((value) => Number.isInteger(value) && value > 0);

export const LayoutRegenerateModal = ({
  open,
  hall,
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

  // Seed via initialValues: StrictMode + preserve=false wipes setFieldsValue-in-effect.
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

      // With a template, drop the layout fields and take the template values.
      const payload: HallPayload = values.template
        ? { name: hall.name, template: values.template }
        : {
            name: hall.name,
            rows: values.rows,
            seats_per_row: values.seats_per_row,
            // Both fields are required: omitting one resets to front and clears aisles.
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
