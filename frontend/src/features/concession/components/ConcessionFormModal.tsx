import { App, Form, Input, InputNumber, Modal, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import type { Combo, CreateComboPayload, UpdateComboPayload } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

/** Mirrors the backend binding tags on CreateComboRequest/UpdateComboRequest. */
const NAME_MIN = 2;
const NAME_MAX = 255;
const DESCRIPTION_MAX = 2000;
const IMAGE_URL_MAX = 512;

interface FormValues {
  name: string;
  description?: string;
  price: number;
  image_url?: string;
  active: boolean;
}

interface ConcessionFormModalProps {
  open: boolean;
  /** null = create, otherwise edit that product. */
  editing: Combo | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throws on failure so the modal stays open and can bind 400/40001 to its inputs. */
  onSubmit: (payload: CreateComboPayload | UpdateComboPayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

/** Create/edit one concession product.
 *
 *  Seeded through `initialValues` + `destroyOnHidden` rather than a mount effect:
 *  under StrictMode antd's `preserve={false}` cleanup runs AFTER an effect's
 *  setFieldsValue and wipes it, which is why the movie modal used to open blank
 *  (see .claude/rules/pages-components.md). */
export const ConcessionFormModal = ({
  open,
  editing,
  confirmLoading,
  onCancel,
  onSubmit,
}: ConcessionFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // Send the whole shape either way: on edit the backend PATCH accepts a
      // full body just as happily as a sparse one, and "changed nothing" is
      // still a real edit the operator may have intended (e.g. re-save).
      await onSubmit({
        name: values.name.trim(),
        description: values.description?.trim() ?? '',
        price: values.price,
        image_url: values.image_url?.trim() ?? '',
        active: values.active,
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
      title={t(editing ? 'concession.editTitle' : 'concession.createTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={560}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={{
          name: editing?.name ?? '',
          description: editing?.description ?? '',
          price: editing?.price ?? 0,
          image_url: editing?.image_url ?? '',
          // New products go on sale by default, matching the backend.
          active: editing?.active ?? true,
        }}
      >
        <Form.Item
          name="name"
          label={t('concession.name')}
          rules={[required, { min: NAME_MIN, max: NAME_MAX }]}
        >
          <Input />
        </Form.Item>

        <Form.Item
          name="description"
          label={t('concession.description')}
          rules={[{ max: DESCRIPTION_MAX }]}
        >
          <Input.TextArea rows={3} />
        </Form.Item>

        <Form.Item
          name="price"
          label={t('concession.price')}
          extra={t('concession.priceHint')}
          rules={[required, { type: 'number', min: 0 }]}
        >
          {/* Typed explicitly: without it TS infers a literal union from min/max. */}
          <InputNumber<number>
            min={0}
            step={1000}
            style={{ width: '100%' }}
            formatter={(value) => `${value ?? ''}`.replace(/\B(?=(\d{3})+(?!\d))/g, '.')}
            parser={(value) => Number((value ?? '').replace(/\./g, ''))}
          />
        </Form.Item>

        {/* The backend validates `url` on create but only `max` on update, so the
            client keeps one consistent rule and lets an empty string through. */}
        <Form.Item
          name="image_url"
          label={t('concession.imageUrl')}
          extra={t('concession.imageUrlHint')}
          rules={[{ max: IMAGE_URL_MAX }, { type: 'url', warningOnly: true }]}
        >
          <Input placeholder="https://..." />
        </Form.Item>

        <Form.Item
          name="active"
          label={t('concession.status')}
          valuePropName="checked"
          extra={t('concession.activeHint')}
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ConcessionFormModal;
