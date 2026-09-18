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
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  MoonOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAppStore } from '@/stores/appStore';
import { useAuthStore } from '@/stores/authStore';
import { PATHS } from '@/routes/paths';
import { useNavItems } from '@/routes/navigation';
import Loading from '@/components/Loading';
import type { AppLanguage } from '@/locales/i18n';
import { brand, textOnBrand } from '@/theme';

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

  // Chi nhung muc role hien tai vao duoc; nguon la routes/navigation.tsx, cung
  // hang so roles ma router dung, nen menu khong the bay ra link 403.
  const navItems = useNavItems();

  const menuItems = useMemo(
    () =>
      navItems.map((item) => ({
        key: item.path,
        icon: item.icon,
        label: <Link to={item.path}>{t(`menu.${item.i18nKey}`)}</Link>,
      })),
    [navItems, t]
  );

  const current = navItems.find((item) => location.pathname.startsWith(item.path));

  const breadcrumbItems = useMemo(
    () => [
      { title: t('common.appName') },
      ...(current ? [{ title: t(`menu.${current.i18nKey}`) }] : []),
    ],
    [current, t]
  );

  const handleLogout = () => {
    logout();
    navigate(PATHS.login, { replace: true });
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* Figma de sider mau trang, ngan voi noi dung bang mot duong vien mong. */}
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={230}
        style={{ borderInlineEnd: `1px solid ${token.colorBorderSecondary}` }}
      >
        <div
          style={{
            height: 56,
            margin: 12,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: token.colorText,
            fontWeight: 700,
            fontSize: collapsed ? 16 : 18,
            letterSpacing: 0.5,
            overflow: 'hidden',
            whiteSpace: 'nowrap',
          }}
        >
          {collapsed ? 'CP' : t('common.appName')}
        </div>
        {/* Luon de theme="light": darkAlgorithm da tu dao bang mau roi. Menu
         * theme="dark" cua antd la dien mao navy #001529 cu, dat tren nen
         * #141414 thi chu muc chua chon gan nhu khong doc duoc. */}
        <Menu mode="inline" selectedKeys={current ? [current.path] : []} items={menuItems} />
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
                {/* Figma: tron mau primary mang chu cai dau, khong phai icon xam. */}
                <Avatar
                  size="small"
                  style={{
                    backgroundColor: brand.base,
                    color: textOnBrand,
                    fontWeight: 600,
                  }}
                >
                  {(user?.full_name ?? user?.email ?? 'U').charAt(0).toUpperCase()}
                </Avatar>
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
