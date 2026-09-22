import { App, DatePicker, Form, Input, InputNumber, Modal, Select, Switch } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type {
  CreateDiscountPayload,
  DiscountCode,
  DiscountKind,
  UpdateDiscountPayload,
} from '@/types';
import { DISCOUNT_KINDS } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

const CODE_MIN = 3;
const CODE_MAX = 32;
const DESCRIPTION_MAX = 2000;

interface FormValues {
  code: string;
  description?: string;
  kind: DiscountKind;
  value: number;
  max_discount?: number | null;
  min_order: number;
  window?: [Dayjs, Dayjs] | null;
  max_uses?: number | null;
  active: boolean;
}

interface DiscountFormModalProps {
  open: boolean;
  /** null = create. On edit, `code` and `kind` are locked: the backend refuses to
   *  change either, because it would rewrite what the code meant for orders that
   *  already used it. */
  editing: DiscountCode | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throws on failure so the modal stays open and can bind 400/40001 to inputs. */
  onSubmit: (payload: CreateDiscountPayload | UpdateDiscountPayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const DiscountFormModal = ({
  open,
  editing,
  confirmLoading,
  onCancel,
  onSubmit,
}: DiscountFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  // Watched so the value field can relabel itself and the cap can hide: the
  // backend REJECTS max_discount on a flat code rather than ignoring it.
  const kind = Form.useWatch('kind', form) ?? editing?.kind ?? 'percent';
  const isPercent = kind === 'percent';

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const [startsAt, endsAt] = values.window ?? [];
      const shared = {
        description: values.description?.trim() ?? '',
        value: values.value,
        // Only a percentage code may carry a cap.
        max_discount: isPercent ? (values.max_discount ?? undefined) : undefined,
        min_order: values.min_order,
        starts_at: startsAt ? startsAt.toISOString() : undefined,
        ends_at: endsAt ? endsAt.toISOString() : undefined,
        max_uses: values.max_uses ?? undefined,
        active: values.active,
      };
      await onSubmit(
        editing ? shared : { ...shared, code: values.code.trim().toUpperCase(), kind: values.kind }
      );
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
      title={t(editing ? 'discount.editTitle' : 'discount.createTitle')}
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
        initialValues={{
          code: editing?.code ?? '',
          description: editing?.description ?? '',
          kind: editing?.kind ?? 'percent',
          value: editing?.value ?? 10,
          max_discount: editing?.max_discount ?? null,
          min_order: editing?.min_order ?? 0,
          window:
            editing?.starts_at && editing?.ends_at
              ? [dayjs(editing.starts_at), dayjs(editing.ends_at)]
              : null,
          max_uses: editing?.max_uses ?? null,
          active: editing?.active ?? true,
        }}
      >
        <Form.Item
          name="code"
          label={t('discount.code')}
          extra={editing ? t('discount.codeLocked') : t('discount.codeHint')}
          rules={editing ? [] : [required, { min: CODE_MIN, max: CODE_MAX }]}
        >
          <Input disabled={Boolean(editing)} style={{ textTransform: 'uppercase' }} />
        </Form.Item>

        <Form.Item
          name="description"
          label={t('discount.description')}
          rules={[{ max: DESCRIPTION_MAX }]}
        >
          <Input.TextArea rows={2} />
        </Form.Item>

        <Form.Item
          name="kind"
          label={t('discount.kind')}
          extra={editing ? t('discount.kindLocked') : undefined}
          rules={[required]}
        >
          <Select<DiscountKind>
            disabled={Boolean(editing)}
            options={DISCOUNT_KINDS.map((k) => ({ value: k, label: t(`discount.kind_${k}`) }))}
          />
        </Form.Item>

        <Form.Item
          name="value"
          label={t(isPercent ? 'discount.valuePercent' : 'discount.valueAmount')}
          rules={[required, { type: 'number', min: 1, max: isPercent ? 100 : undefined }]}
        >
          <InputNumber<number>
            min={1}
            max={isPercent ? 100 : undefined}
            style={{ width: '100%' }}
          />
        </Form.Item>

        {isPercent ? (
          <Form.Item
            name="max_discount"
            label={t('discount.maxDiscount')}
            extra={t('discount.maxDiscountHint')}
            rules={[{ type: 'number', min: 1 }]}
          >
            <InputNumber<number> min={1} step={1000} style={{ width: '100%' }} />
          </Form.Item>
        ) : null}

        <Form.Item
          name="min_order"
          label={t('discount.minOrder')}
          extra={t('discount.minOrderHint')}
          rules={[{ type: 'number', min: 0 }]}
        >
          <InputNumber<number> min={0} step={10000} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="window" label={t('discount.window')} extra={t('discount.windowHint')}>
          <DatePicker.RangePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="max_uses"
          label={t('discount.maxUses')}
          extra={t('discount.maxUsesHint')}
          rules={[{ type: 'number', min: 1 }]}
        >
          <InputNumber<number> min={1} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="active"
          label={t('discount.status')}
          valuePropName="checked"
          extra={t('discount.activeHint')}
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default DiscountFormModal;
