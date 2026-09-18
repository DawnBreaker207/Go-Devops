import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Alert, Button, Card, Form, Input, Typography } from 'antd';
import { LockOutlined, MailOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';
import { useLandingPath } from '@/routes/navigation';
import type { ApiError, LoginRequest } from '@/types';

interface LocationState {
  from?: string;
}

export const LoginPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const landing = useLandingPath();
  const location = useLocation();
  const login = useAuthStore((s) => s.login);

  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleSubmit = async (values: LoginRequest) => {
    setSubmitting(true);
    setErrorMessage(null);
    try {
      await login(values);
      const from = (location.state as LocationState | null)?.from;
      // Khong co dinh /dashboard: khach khong vao duoc trang van hanh nao va
      // se roi thang vao man 403. useLandingPath tra ve dung noi role do vao duoc.
      navigate(from ?? landing, { replace: true });
    } catch (error) {
      setErrorMessage((error as ApiError).message || t('auth.loginFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
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
    </Card>
  );
};

export default LoginPage;
