import {
  App,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  TimePicker,
} from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type {
  CreatePricingRulePayload,
  PricingAdjustKind,
  PricingRule,
  UpdatePricingRulePayload,
} from '@/types';
import { PRICING_ADJUST_KINDS } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { API_DATE_FORMAT } from '@/utils/format';

const NAME_MAX = 255;
const TIME_FORMAT = 'HH:mm';

/** Sun..Sat, matching the backend's 0-6 day_of_week (Go time.Weekday). */
const DAYS_OF_WEEK = [0, 1, 2, 3, 4, 5, 6] as const;

interface FormValues {
  name: string;
  day_of_week?: number | null;
  time_window?: [Dayjs, Dayjs] | null;
  specific_date?: Dayjs | null;
  adjust_kind: PricingAdjustKind;
  adjust_value: number;
  priority: number;
}

interface PricingRuleFormModalProps {
  open: boolean;
  entity: PricingRule | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throws on failure so the modal stays open and can bind 400/40001 to inputs. */
  onSubmit: (payload: CreatePricingRulePayload | UpdatePricingRulePayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const PricingRuleFormModal = ({
  open,
  entity,
  confirmLoading,
  onCancel,
  onSubmit,
}: PricingRuleFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const [startTime, endTime] = values.time_window ?? [];
      const shared = {
        name: values.name.trim(),
        day_of_week: values.day_of_week ?? undefined,
        start_time: startTime ? startTime.format(TIME_FORMAT) : undefined,
        end_time: endTime ? endTime.format(TIME_FORMAT) : undefined,
        specific_date: values.specific_date
          ? values.specific_date.format(API_DATE_FORMAT)
          : undefined,
        adjust_kind: values.adjust_kind,
        adjust_value: values.adjust_value,
        priority: values.priority,
        active: entity?.active ?? false,
      };
      await onSubmit(shared);
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
      title={t(entity ? 'pricing.editRuleTitle' : 'pricing.createRuleTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={640}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={{
          name: entity?.name ?? '',
          day_of_week: entity?.day_of_week ?? null,
          time_window:
            entity?.start_time && entity?.end_time
              ? [dayjs(entity.start_time, TIME_FORMAT), dayjs(entity.end_time, TIME_FORMAT)]
              : null,
          specific_date: entity?.specific_date
            ? dayjs(entity.specific_date, API_DATE_FORMAT)
            : null,
          adjust_kind: entity?.adjust_kind ?? 'percent',
          adjust_value: entity?.adjust_value ?? 0,
          priority: entity?.priority ?? 0,
        }}
      >
        <Form.Item name="name" label={t('pricing.ruleName')} rules={[required, { max: NAME_MAX }]}>
          <Input />
        </Form.Item>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="day_of_week"
              label={t('pricing.dayOfWeek')}
              extra={t('pricing.dayOfWeekHint')}
            >
              <Select
                allowClear
                placeholder={t('pricing.dayOfWeekAny')}
                options={DAYS_OF_WEEK.map((day) => ({
                  value: day,
                  label: t(`pricing.dow_${day}`),
                }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="specific_date"
              label={t('pricing.specificDate')}
              extra={t('pricing.specificDateHint')}
            >
              <DatePicker format="DD/MM/YYYY" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="time_window"
              label={t('pricing.timeWindow')}
              extra={t('pricing.timeWindowHint')}
            >
              <TimePicker.RangePicker format={TIME_FORMAT} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="priority"
              label={t('pricing.priority')}
              extra={t('pricing.priorityHint')}
              rules={[required, { type: 'number', min: 0 }]}
            >
              <InputNumber<number> min={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="adjust_kind" label={t('pricing.adjustKind')} rules={[required]}>
              <Select<PricingAdjustKind>
                options={PRICING_ADJUST_KINDS.map((kind) => ({
                  value: kind,
                  label: t(`pricing.adjustKind_${kind}`),
                }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="adjust_value"
              label={t('pricing.adjustValue')}
              extra={t('pricing.adjustValueHint')}
              rules={[required, { type: 'number' }]}
            >
              <InputNumber<number> style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
};

export default PricingRuleFormModal;
