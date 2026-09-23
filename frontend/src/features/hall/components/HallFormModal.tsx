import {
  Alert,
  App,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Switch,
  Typography,
} from 'antd';
import { useTranslation } from 'react-i18next';
import type {
  Hall,
  HallPayload,
  HallTemplateName,
  ScreenPosition,
  SeatType,
  UpdateHallPayload,
} from '@/types';
import { SCREEN_POSITIONS, SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { formatVND } from '@/utils/format';
import { useHallTemplates } from '../hooks/useHalls';
import { MAX_ROWS, MAX_SEAT_PRICE, MAX_SEATS_PER_ROW, MIN_SEAT_PRICE } from '../constants';

interface FormValues {
  name: string;
  template?: HallTemplateName | '';
  rows?: number;
  seats_per_row?: number;
  screen_position: ScreenPosition;
  aisle_after_cols: string;
  active?: boolean;
  prices?: Record<SeatType, number>;
}

interface HallFormModalProps {
  open: boolean;
  /** null = create (full form); set = edit (only the 4 UpdateHallRequest fields). */
  hall: Hall | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throw on failure so the modal stays open and maps errors onto the right inputs. */
  onCreate: (payload: HallPayload) => Promise<void>;
  onUpdate: (payload: UpdateHallPayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

const parseAisles = (raw: string): number[] =>
  raw
    .split(',')
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((value) => Number.isInteger(value) && value > 0);

export const HallFormModal = ({
  open,
  hall,
  confirmLoading,
  onCancel,
  onCreate,
  onUpdate,
}: HallFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const templates = useHallTemplates();

  const isEdit = hall !== null;
  const template = Form.useWatch('template', form);
  const usingTemplate = Boolean(template);

  // Seed via initialValues, never via setFieldsValue in an effect.
  const initialValues: Partial<FormValues> = hall
    ? {
        name: hall.name,
        screen_position: hall.screen_position,
        aisle_after_cols: hall.aisle_after_cols.join(', '),
        active: hall.active,
      }
    : {
        template: '',
        rows: 10,
        seats_per_row: 12,
        screen_position: 'front',
        aisle_after_cols: '',
      };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();

      if (hall) {
        // PUT /admin/halls/:id is PATCH in semantics: any omitted field is kept
        // as-is. But aisle_after_cols is NOT a pointer in Go, so sending []
        // CLEARS all aisles - which is intended when the user empties the input.
        await onUpdate({
          name: values.name,
          screen_position: values.screen_position,
          aisle_after_cols: parseAisles(values.aisle_after_cols),
          active: values.active,
        });
        return;
      }

      await onCreate({
        name: values.name,
        screen_position: values.screen_position,
        aisle_after_cols: parseAisles(values.aisle_after_cols),
        prices: values.prices as Record<SeatType, number>,
        ...(values.template
          ? { template: values.template }
          : { rows: values.rows, seats_per_row: values.seats_per_row }),
      });
    } catch (error) {
      if (isFormValidationError(error)) return;
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const required = { required: true, message: t('common.requiredField') };

  return (
    <Modal
      open={open}
      title={isEdit ? t('hall.editTitle') : t('hall.createTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={620}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        <Form.Item name="name" label={t('hall.name')} rules={[required, { max: 255 }]}>
          <Input placeholder={t('hall.namePlaceholder')} />
        </Form.Item>

        {isEdit ? (
          <>
            {/* Editing a hall NEVER touches seats. Changing the grid is a separate
                operation, done in the seat-map screen. */}
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={t('hall.editScopeNote')}
            />
            <Form.Item
              name="active"
              label={t('hall.active')}
              valuePropName="checked"
              extra={t('hall.activeHint')}
            >
              <Switch />
            </Form.Item>
          </>
        ) : (
          <>
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

            {usingTemplate ? null : (
              <>
                <Form.Item name="rows" label={t('hall.rows')} rules={[required]}>
                  <InputNumber min={1} max={MAX_ROWS} style={{ width: '100%' }} />
                </Form.Item>
                <Form.Item name="seats_per_row" label={t('hall.seatsPerRow')} rules={[required]}>
                  <InputNumber min={1} max={MAX_SEATS_PER_ROW} style={{ width: '100%' }} />
                </Form.Item>
              </>
            )}
          </>
        )}

        <Form.Item name="screen_position" label={t('hall.screenPosition')}>
          <Select
            options={SCREEN_POSITIONS.map((value) => ({ value, label: t(`hall.screen_${value}`) }))}
          />
        </Form.Item>

        <Form.Item name="aisle_after_cols" label={t('hall.aisles')} extra={t('hall.aislesHint')}>
          <Input placeholder="4, 10" />
        </Form.Item>

        {isEdit ? null : (
          <>
            <Divider orientation="left" plain>
              {t('hall.prices')}
            </Divider>
            {/* Backend requires ALL 4 price types, all > 0, right at creation; a missing
                type is 400/40001 with details {seat_type}. There is no way to create
                a hall with an empty price table. */}
            <Typography.Paragraph type="secondary" style={{ fontSize: 12 }}>
              {t('hall.pricesRequiredHint')}
            </Typography.Paragraph>
            {SEAT_TYPES.map((type) => (
              <Form.Item
                key={type}
                name={['prices', type]}
                label={t(`hall.seatType_${type}`)}
                rules={[required]}
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
          </>
        )}
      </Form>
    </Modal>
  );
};

export default HallFormModal;
