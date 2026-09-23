import { Suspense, useMemo } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import {
  Avatar,
  Breadcrumb,
  Button,
  Dropdown,
  Layout,
  Menu,
  Select,
  Space,
  Typography,
  theme as antdTheme,
} from 'antd';
import {
  BulbOutlined,
  CalendarOutlined,
  DashboardOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  MoonOutlined,
  ScheduleOutlined,
  UserOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAppStore } from '@/stores/appStore';
import { useAuthStore } from '@/stores/authStore';
import { PATHS } from '@/routes/paths';
import Loading from '@/components/Loading';
import type { AppLanguage } from '@/locales/i18n';

const { Header, Sider, Content } = Layout;

export const MainLayout = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { token } = antdTheme.useToken();

  const collapsed = useAppStore((s) => s.siderCollapsed);
  const toggleSider = useAppStore((s) => s.toggleSider);
  const themeMode = useAppStore((s) => s.theme);
  const toggleTheme = useAppStore((s) => s.toggleTheme);
  const language = useAppStore((s) => s.language);
  const setLanguage = useAppStore((s) => s.setLanguage);

  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  const menuItems = useMemo(
    () => [
      {
        key: PATHS.dashboard,
        icon: <DashboardOutlined />,
        label: <Link to={PATHS.dashboard}>{t('menu.dashboard')}</Link>,
      },
      {
        key: PATHS.movies,
        icon: <VideoCameraOutlined />,
        label: <Link to={PATHS.movies}>{t('menu.movies')}</Link>,
      },
      {
        key: PATHS.showtimes,
        icon: <CalendarOutlined />,
        label: <Link to={PATHS.showtimes}>{t('menu.showtimes')}</Link>,
      },
      {
        key: PATHS.bookings,
        icon: <ScheduleOutlined />,
        label: <Link to={PATHS.bookings}>{t('menu.bookings')}</Link>,
      },
    ],
    [t]
  );

  const selectedKey =
    menuItems.find((item) => location.pathname.startsWith(item.key))?.key ?? PATHS.dashboard;

  const breadcrumbItems = useMemo(() => {
    const current = menuItems.find((item) => item.key === selectedKey);
    return [
      { title: t('common.appName') },
      { title: current ? t(`menu.${selectedKey.replace('/', '') || 'dashboard'}`) : '' },
    ];
  }, [menuItems, selectedKey, t]);

  const handleLogout = () => {
    logout();
    navigate(PATHS.login, { replace: true });
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed} theme="dark" width={230}>
        <div
          style={{
            height: 56,
            margin: 12,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontWeight: 700,
            fontSize: collapsed ? 16 : 18,
            letterSpacing: 0.5,
            overflow: 'hidden',
            whiteSpace: 'nowrap',
          }}
        >
          {collapsed ? 'CP' : t('common.appName')}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[selectedKey]} items={menuItems} />
      </Sider>

      <Layout>
        <Header
          style={{
            padding: '0 16px',
            background: token.colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Button
            type="text"
            aria-label="toggle-sider"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={toggleSider}
          />

          <Space size="middle">
            <Select<AppLanguage>
              size="small"
              value={language}
              onChange={setLanguage}
              style={{ width: 88 }}
              options={[
                { value: 'vi', label: 'Tiếng Việt' },
                { value: 'en', label: 'English' },
              ]}
            />
            <Button
              type="text"
              aria-label="toggle-theme"
              icon={themeMode === 'light' ? <MoonOutlined /> : <BulbOutlined />}
              onClick={toggleTheme}
            />
            <Dropdown
              menu={{
                items: [
                  { key: 'profile', icon: <UserOutlined />, label: t('common.profile') },
                  { type: 'divider' },
                  {
                    key: 'logout',
                    icon: <LogoutOutlined />,
                    danger: true,
                    label: t('common.logout'),
                    onClick: handleLogout,
                  },
                ],
              }}
            >
              <Space style={{ cursor: 'pointer' }}>
                <Avatar size="small" icon={<UserOutlined />} />
                <Typography.Text>{user?.full_name ?? user?.email ?? 'User'}</Typography.Text>
              </Space>
            </Dropdown>
          </Space>
        </Header>

        <Content style={{ margin: 16 }}>
          <Breadcrumb items={breadcrumbItems} style={{ marginBottom: 12 }} />
          <div
            style={{
              padding: 16,
              background: token.colorBgContainer,
              borderRadius: token.borderRadiusLG,
              minHeight: 'calc(100vh - 160px)',
            }}
          >
            <Suspense fallback={<Loading />}>
              <Outlet />
            </Suspense>
          </div>
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
