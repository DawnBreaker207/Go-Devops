import { Alert, App, Form, Input, Modal, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import type { CreateUserPayload, UserRole } from '@/types';
import { CREATABLE_ROLES } from '@/types';
import { errorMessage } from '@/utils/error';
import { applyApiFieldErrors } from '@/utils/form';

/** Tran cua bcrypt, cung la binding max=72 ben backend. */
const PASSWORD_MAX = 72;
const PASSWORD_MIN = 6;

interface FormValues {
  email: string;
  password: string;
  full_name: string;
  role: UserRole;
}

interface UserFormModalProps {
  open: boolean;
  confirmLoading: boolean;
  onCancel: () => void;
  /** Nem loi ra thi modal giu nguyen va tu gan loi vao dung o nhap. */
  onSubmit: (payload: CreateUserPayload) => Promise<void>;
}

const isFormValidationError = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'errorFields' in error;

/**
 * Chi TAO tai khoan. Khong co man sua ho so nguoi khac: backend chi cho doi
 * `active` va `role` qua PATCH, con ten/email/dien thoai thi chinh chu doi lay
 * qua `PUT /users/me`.
 */
export const UserFormModal = ({ open, confirmLoading, onCancel, onSubmit }: UserFormModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [form] = Form.useForm<FormValues>();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      await onSubmit({
        email: values.email.trim(),
        password: values.password,
        full_name: values.full_name.trim(),
        role: values.role,
      });
    } catch (error) {
      if (isFormValidationError(error)) return;
      // Trung email la 409/40900 khong kem details, nen roi xuong toast.
      if (applyApiFieldErrors(form, error)) return;
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const required = { required: true, message: t('common.requiredField') };

  return (
    <Modal
      open={open}
      title={t('user.createTitle')}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={520}
    >
      {/* Backend chan tao tai khoan khach o day (binding oneof=staff admin):
          khach tu dang ky qua /auth/register. */}
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message={t('user.createScopeNote')}
      />

      <Form<FormValues>
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={{ role: 'staff' }}
      >
        <Form.Item
          name="email"
          label={t('user.email')}
          rules={[required, { type: 'email', message: t('user.emailInvalid') }, { max: 255 }]}
        >
          <Input autoComplete="off" />
        </Form.Item>

        <Form.Item
          name="full_name"
          label={t('user.fullName')}
          rules={[required, { min: 2, max: 255 }]}
        >
          <Input />
        </Form.Item>

        <Form.Item
          name="password"
          label={t('user.password')}
          extra={t('user.passwordHint')}
          rules={[required, { min: PASSWORD_MIN, max: PASSWORD_MAX }]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>

        <Form.Item name="role" label={t('user.role')} rules={[required]}>
          <Select
            options={CREATABLE_ROLES.map((role) => ({
              value: role,
              label: t(`user.role_${role}`),
            }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default UserFormModal;
