import { Alert, App, Form, Input, Modal, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import type { CloneHallPayload, Hall } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';
import { useCloneHall } from '../hooks/useHalls';

interface FormValues {
  name: string;
  copy_prices: boolean;
}

interface CloneHallModalProps {
  open: boolean;
  hall: Hall | null;
  onCancel: () => void;
  onDone: () => void;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

export const CloneHallModal = ({ open, hall, onCancel, onDone }: CloneHallModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const clone = useCloneHall();

  // copy_prices defaults to false on the backend, but TRUE here: a hall
  // without prices vanishes from every customer list with no warning.
  const initialValues: FormValues = {
    name: hall ? t('hall.cloneNameSuggestion', { name: hall.name }) : '',
    copy_prices: true,
  };

  const handleOk = async () => {
    if (!hall) return;
    try {
      const values = await form.validateFields();
      const payload: CloneHallPayload = { name: values.name, copy_prices: values.copy_prices };
      await clone.mutateAsync({ id: hall.id, payload });
      message.success(t('hall.cloneSuccess'));
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
      title={hall ? t('hall.cloneTitle', { name: hall.name }) : t('hall.clone')}
      okText={t('hall.clone')}
      cancelText={t('common.cancel')}
      confirmLoading={clone.isPending}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={480}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        <Form.Item
          name="name"
          label={t('hall.name')}
          rules={[{ required: true, message: t('common.requiredField') }, { max: 255 }]}
        >
          <Input />
        </Form.Item>
        <Form.Item name="copy_prices" label={t('hall.copyPrices')} valuePropName="checked">
          <Switch />
        </Form.Item>
        <Alert type="info" showIcon message={t('hall.copyPricesHint')} />
      </Form>
    </Modal>
  );
};

export default CloneHallModal;
