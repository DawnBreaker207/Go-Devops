import { useState } from 'react';
import { App, Button, Form, Image, Input, InputNumber, Modal, Upload } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  POSTER_ACCEPT,
  POSTER_MAX_BYTES,
  type Combo,
  type CreateComboPayload,
  type UpdateComboPayload,
} from '@/types';
import { mediaApi } from '@/api/media.api';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

const NAME_MIN = 2;
const NAME_MAX = 255;
const DESCRIPTION_MAX = 2000;
const IMAGE_URL_MAX = 512;

interface FormValues {
  name: string;
  description?: string;
  price: number;
  image_url?: string;
}

interface ConcessionFormModalProps {
  open: boolean;
  editing: Combo | null;
  confirmLoading: boolean;
  onCancel: () => void;
  onSubmit: (payload: CreateComboPayload | UpdateComboPayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

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
  const [uploading, setUploading] = useState(false);

  const watchedImageUrl = Form.useWatch('image_url', form);

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const result = await mediaApi.uploadPoster(file);
      form.setFieldValue('image_url', result.url);
      await form.validateFields(['image_url']);
      message.success(t('concession.imageUploaded'));
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    } finally {
      setUploading(false);
    }
  };

  const uploadProps = {
    accept: POSTER_ACCEPT.join(','),
    showUploadList: false as const,
    beforeUpload: (file: File) => {
      if (!POSTER_ACCEPT.includes(file.type as (typeof POSTER_ACCEPT)[number])) {
        message.error(t('concession.imageTypeInvalid'));
        return Upload.LIST_IGNORE;
      }
      if (file.size > POSTER_MAX_BYTES) {
        message.error(t('concession.imageTooLarge'));
        return Upload.LIST_IGNORE;
      }
      void handleUpload(file);
      return false;
    },
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      await onSubmit({
        name: values.name.trim(),
        description: values.description?.trim() ?? '',
        price: values.price,
        image_url: values.image_url?.trim() ?? '',
        // Not exposed here: new products start off sale, existing ones keep whatever
        // the list page's own toggle last set (see ConcessionsPage.toggleActive).
        active: editing?.active ?? false,
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
        }}
      >
        <Form.Item
          name="name"
          label={t('concession.name')}
          rules={[required, { min: NAME_MIN, max: NAME_MAX }]}
        >
          <Input maxLength={NAME_MAX} showCount />
        </Form.Item>

        <Form.Item
          name="description"
          label={t('concession.description')}
          rules={[{ max: DESCRIPTION_MAX }]}
        >
          <Input.TextArea rows={3} maxLength={DESCRIPTION_MAX} showCount />
        </Form.Item>

        <Form.Item
          name="price"
          label={t('concession.price')}
          extra={t('concession.priceHint')}
          rules={[required, { type: 'number', min: 0 }]}
        >
          <InputNumber<number>
            min={0}
            step={1000}
            style={{ width: '100%' }}
            formatter={(value) => `${value ?? ''}`.replace(/\B(?=(\d{3})+(?!\d))/g, '.')}
            parser={(value) => Number((value ?? '').replace(/\./g, ''))}
          />
        </Form.Item>

        <Form.Item
          name="image_url"
          label={t('concession.imageUrl')}
          extra={t('concession.imageUrlHint')}
          rules={[{ max: IMAGE_URL_MAX }, { type: 'url', warningOnly: true }]}
        >
          <Input
            placeholder="https://..."
            maxLength={IMAGE_URL_MAX}
            showCount
            addonAfter={
              <Upload {...uploadProps}>
                <Button
                  type="text"
                  size="small"
                  icon={<UploadOutlined />}
                  loading={uploading}
                  aria-label={t('concession.imageUpload')}
                />
              </Upload>
            }
          />
        </Form.Item>

        {watchedImageUrl ? (
          <Form.Item label={t('concession.imagePreview')}>
            <Image
              src={watchedImageUrl}
              alt=""
              width={120}
              height={120}
              style={{ objectFit: 'cover', borderRadius: 8 }}
            />
          </Form.Item>
        ) : null}
      </Form>
    </Modal>
  );
};

export default ConcessionFormModal;
