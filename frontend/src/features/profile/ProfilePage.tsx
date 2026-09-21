import {
  App,
  Avatar,
  Button,
  Card,
  Empty,
  Form,
  Input,
  List,
  Spin,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { IdcardOutlined, LaptopOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';
import { useRevokeSession, useSessions, useUpdateProfile } from '@/hooks/useAccount';
import { errorMessage, fieldErrorsOf } from '@/utils/error';
import { formatDateTime } from '@/utils/format';
import { brand, textOnBrand } from '@/theme';

const { Title, Text } = Typography;

interface ProfileFormValues {
  full_name: string;
  phone?: string;
}

const ProfileTab = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const user = useAuthStore((s) => s.user);
  const updateProfile = useUpdateProfile();
  const [form] = Form.useForm<ProfileFormValues>();

  if (!user) return null;

  const submit = async (values: ProfileFormValues) => {
    try {
      await updateProfile.mutateAsync({
        full_name: values.full_name.trim(),
        phone: (values.phone ?? '').trim(),
      });
      message.success(t('common.updateSuccess'));
    } catch (error) {
      const details = fieldErrorsOf(error);
      if (details) {
        form.setFields(
          Object.entries(details).map(([name, msg]) => ({
            name: name as keyof ProfileFormValues,
            errors: [msg],
          }))
        );
      } else {
        message.error(errorMessage(error, t('common.somethingWrong')));
      }
    }
  };

  return (
    <Form
      form={form}
      layout="vertical"
      style={{ maxWidth: 420 }}
      initialValues={{ full_name: user.full_name, phone: user.phone ?? '' }}
      onFinish={submit}
    >
      <Form.Item label={t('user.email')}>
        <Input value={user.email} disabled />
      </Form.Item>
      <Form.Item name="full_name" label={t('user.fullName')} rules={[{ required: true, min: 2 }]}>
        <Input />
      </Form.Item>
      <Form.Item name="phone" label={t('customer.accountPhone')}>
        <Input />
      </Form.Item>
      <Button type="primary" htmlType="submit" loading={updateProfile.isPending}>
        {t('common.save')}
      </Button>
    </Form>
  );
};

const SessionsTab = () => {
  const { t } = useTranslation();
  const { data, isLoading } = useSessions();
  const revoke = useRevokeSession();

  return (
    <List
      loading={isLoading}
      dataSource={data ?? []}
      locale={{ emptyText: <Empty description={t('customer.accountNoSessions')} /> }}
      renderItem={(session) => (
        <List.Item
          actions={
            !session.is_current
              ? [
                  <Button
                    key="revoke"
                    type="link"
                    danger
                    loading={revoke.isPending}
                    onClick={() => revoke.mutate(session.id)}
                  >
                    {t('customer.accountSignOutDevice')}
                  </Button>,
                ]
              : undefined
          }
        >
          <List.Item.Meta
            title={
              <>
                {session.user_agent || t('customer.accountUnknownDevice')}
                {session.is_current ? (
                  <Tag color="blue" style={{ marginLeft: 8 }}>
                    {t('customer.accountCurrentDevice')}
                  </Tag>
                ) : null}
              </>
            }
            description={`${t('customer.accountSessionLastUsed')}: ${
              session.last_used_at ? formatDateTime(session.last_used_at) : '-'
            }`}
          />
        </List.Item>
      )}
    />
  );
};

/** Operator "Profile" (avatar menu in MainLayout). Same card + left-icon-tab frame as the customer AccountPage but antd Card/Tabs per operator convention (no customer INK tokens; those only exist under CustomerLayout). 2 tabs only: Profile (PUT /users/me, same role-agnostic endpoint) and Devices (/users/me/sessions). Skips transactions/notifications/membership: buyer concepts, not operator ones. */
export const ProfilePage = () => {
  const { t } = useTranslation();
  const user = useAuthStore((s) => s.user);

  if (!user) return <Spin />;

  const initials = (user.full_name || user.email).charAt(0).toUpperCase();

  return (
    <div>
      <div className="mb-5 flex items-center gap-3.5">
        <Avatar
          size={52}
          style={{ backgroundColor: brand.base, color: textOnBrand, fontWeight: 700, fontSize: 20 }}
        >
          {initials}
        </Avatar>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            {user.full_name}
          </Title>
          <Text type="secondary">{user.email}</Text>
        </div>
      </div>

      <Card styles={{ body: { padding: 0 } }}>
        <Tabs
          tabPosition="left"
          items={[
            {
              key: 'profile',
              label: (
                <span>
                  <IdcardOutlined /> {t('customer.accountTab_profile')}
                </span>
              ),
              children: (
                <div style={{ padding: 24 }}>
                  <ProfileTab />
                </div>
              ),
            },
            {
              key: 'sessions',
              label: (
                <span>
                  <LaptopOutlined /> {t('customer.accountTab_sessions')}
                </span>
              ),
              children: (
                <div style={{ padding: 24 }}>
                  <SessionsTab />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default ProfilePage;
