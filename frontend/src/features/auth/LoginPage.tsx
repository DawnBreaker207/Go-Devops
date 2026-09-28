import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Alert, Button, Card, Form, Input, Typography } from 'antd';
import { LockOutlined, MailOutlined, VideoCameraOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { performLogin } from './loginFlow';
import { brand, textOnBrand } from '@/theme';
import type { LoginRequest } from '@/types';

interface LocationState {
  from?: string;
}

export const LoginPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();

  const [form] = Form.useForm<LoginRequest>();
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleSubmit = async (values: LoginRequest) => {
    setSubmitting(true);
    setErrorMessage(null);
    const result = await performLogin(values, t);
    setSubmitting(false);
    if (result.ok) {
      const from = (location.state as LocationState | null)?.from;
      // Never hardcode /dashboard: customers 403 there, resolve the landing path by role.
      navigate(from ?? result.landingPath, { replace: true });
    } else if (result.fieldErrors) {
      // A 400/40001 the client rules missed lands on the matching input.
      form.setFields(
        Object.entries(result.fieldErrors).map(([name, message]) => ({
          name: name as keyof LoginRequest,
          errors: [message],
        }))
      );
    } else {
      setErrorMessage(result.message);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 20 }}>
      <span
        aria-hidden="true"
        style={{
          display: 'flex',
          height: 56,
          width: 56,
          alignItems: 'center',
          justifyContent: 'center',
          borderRadius: 16,
          background: brand.base,
          color: textOnBrand,
          boxShadow: `0 12px 24px -8px ${brand.base}66`,
        }}
      >
        <VideoCameraOutlined style={{ fontSize: 26 }} />
      </span>

      <Card style={{ width: 400, maxWidth: '92vw' }} variant="borderless">
        <Typography.Title level={3} style={{ marginBottom: 4, textAlign: 'center' }}>
          {t('auth.loginTitle')}
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ textAlign: 'center' }}>
          {t('auth.loginSubtitle')}
        </Typography.Paragraph>

        {errorMessage ? (
          <Alert type="error" showIcon message={errorMessage} style={{ marginBottom: 16 }} />
        ) : null}

        <Form<LoginRequest>
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          autoComplete="off"
          requiredMark={false}
        >
          <Form.Item
            name="email"
            label={t('auth.email')}
            rules={[
              { required: true, message: t('auth.emailRequired') },
              { type: 'email', message: t('auth.emailInvalid') },
            ]}
          >
            <Input prefix={<MailOutlined />} placeholder="admin@cinema.local" size="large" />
          </Form.Item>

          <Form.Item
            name="password"
            label={t('auth.password')}
            rules={[
              { required: true, message: t('auth.passwordRequired') },
              { min: 6, message: t('auth.passwordMin') },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="••••••" size="large" />
          </Form.Item>

          <Button type="primary" htmlType="submit" size="large" block loading={submitting}>
            {t('auth.login')}
          </Button>
        </Form>

        {/* khong co quen mat khau staff nen giu 1 dong huong dan. */}
        <Typography.Paragraph
          type="secondary"
          style={{ marginTop: 16, marginBottom: 0, textAlign: 'center', fontSize: 13 }}
        >
          {t('auth.loginHint')}
        </Typography.Paragraph>
      </Card>
    </div>
  );
};

export default LoginPage;
