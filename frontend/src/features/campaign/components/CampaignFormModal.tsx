import { App, DatePicker, Form, Input, InputNumber, Modal } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { Campaign, CreateCampaignPayload, UpdateCampaignPayload } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

const NAME_MAX = 255;
const DESCRIPTION_MAX = 2000;

interface FormValues {
  name: string;
  description?: string;
  window: [Dayjs, Dayjs] | null;
  per_user_limit: number;
}

interface CampaignFormModalProps {
  open: boolean;
  editing: Campaign | null;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Throws on failure so the modal stays open and can bind 400/40001 to inputs. */
  onSubmit: (payload: CreateCampaignPayload | UpdateCampaignPayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const CampaignFormModal = ({
  open,
  editing,
  confirmLoading,
  onCancel,
  onSubmit,
}: CampaignFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const [startsAt, endsAt] = values.window ?? [];
      const shared = {
        name: values.name.trim(),
        description: values.description?.trim() ?? '',
        starts_at: startsAt ? startsAt.toISOString() : undefined,
        ends_at: endsAt ? endsAt.toISOString() : undefined,
        per_user_limit: values.per_user_limit,
        // Not editable here: a new campaign starts inactive (a draft); toggling an existing one
        // is the table's own switch (CampaignsPage.toggleActive), not this form.
        active: editing?.active ?? false,
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
      title={t(editing ? 'campaign.editTitle' : 'campaign.createTitle')}
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
          name: editing?.name ?? '',
          description: editing?.description ?? '',
          window:
            editing?.starts_at && editing?.ends_at
              ? [dayjs(editing.starts_at), dayjs(editing.ends_at)]
              : null,
          per_user_limit: editing?.per_user_limit ?? 1,
        }}
      >
        <Form.Item name="name" label={t('campaign.name')} rules={[required, { max: NAME_MAX }]}>
          <Input />
        </Form.Item>

        <Form.Item name="window" label={t('campaign.window')} rules={[required]}>
          <DatePicker.RangePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="per_user_limit"
          label={t('campaign.perUserLimit')}
          extra={t('campaign.perUserLimitHint')}
          rules={[required, { type: 'number', min: 1 }]}
        >
          <InputNumber<number> min={1} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="description"
          label={t('campaign.description')}
          rules={[{ max: DESCRIPTION_MAX }]}
        >
          <Input.TextArea rows={2} autoSize={{ minRows: 2, maxRows: 4 }} />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default CampaignFormModal;
