import { Suspense, useMemo } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { Breadcrumb, Layout, Menu, theme as antdTheme, type MenuProps } from 'antd';
import { HomeOutlined, RightOutlined, VideoCameraOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAppStore } from '@/stores/appStore';
import { PATHS } from '@/routes/paths';
import { useNavItems, type NavGroup } from '@/routes/navigation';
import Loading from '@/components/Loading';
import AvatarSwitcher from './AvatarSwitcher';
import { brand, textOnBrand } from '@/theme';

const { Header, Sider, Content, Footer } = Layout;

/** 'overview' (Dashboard) stands alone with no group title. */
const GROUP_ORDER: NavGroup[] = ['operations', 'management'];

export const MainLayout = () => {
  const { t } = useTranslation();
  const location = useLocation();
  const { token } = antdTheme.useToken();

  const collapsed = useAppStore((s) => s.siderCollapsed);
  const toggleSider = useAppStore((s) => s.toggleSider);
  const themeMode = useAppStore((s) => s.theme);

  // Filter by role with the same router constants so the menu never shows a 403 link.
  const navItems = useNavItems();

  // Grouped sider: overview stands alone first, empty groups dropped.
  const menuItems = useMemo<MenuProps['items']>(() => {
    const overviewItems = navItems
      .filter((item) => item.group === 'overview')
      .map((item) => ({
        key: item.path,
        icon: item.icon,
        label: <Link to={item.path}>{t(`menu.${item.i18nKey}`)}</Link>,
      }));

    const groupedItems = GROUP_ORDER.map((group) => {
      const items = navItems.filter((item) => item.group === group);
      if (items.length === 0) return null;
      const groupLabel = `${group.charAt(0).toUpperCase()}${group.slice(1)}`;
      return {
        key: group,
        type: 'group' as const,
        label: t(`menu.group${groupLabel}`),
        children: items.map((item) => ({
          key: item.path,
          icon: item.icon,
          label: <Link to={item.path}>{t(`menu.${item.i18nKey}`)}</Link>,
        })),
      };
    }).filter((group) => group !== null);

    return [...overviewItems, ...groupedItems];
  }, [navItems, t]);

  const current = navItems.find((item) => location.pathname.startsWith(item.path));

  const breadcrumbItems = useMemo(
    () => [
      {
        title: (
          <Link to={PATHS.dashboard} style={{ display: 'inline-flex', alignItems: 'center' }}>
            <HomeOutlined />
          </Link>
        ),
      },
      ...(current ? [{ title: t(`menu.${current.i18nKey}`) }] : []),
    ],
    [current, t]
  );

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* 280px sider with the built-in bottom trigger. */}
      <Sider
        // Sider `theme` is independent of the ConfigProvider light/dark algorithm (dark by default),
        // so set "light" explicitly.
        theme="light"
        collapsible
        collapsed={collapsed}
        // `toggleSider` only flips state: compare before calling because antd fires
        // onCollapse(true, 'responsive') even when already collapsed, which would reopen it.
        onCollapse={(value) => {
          if (value !== collapsed) toggleSider();
        }}
        breakpoint="lg"
        width={280}
        className="sticky top-0 h-screen overflow-auto"
        style={{
          borderInlineEnd: `1px solid ${token.colorBorderSecondary}`,
          boxShadow: token.boxShadowTertiary,
        }}
      >
        <div
          className="sticky top-0 z-10 flex h-20 items-center gap-3 overflow-hidden px-4 whitespace-nowrap"
          style={{
            justifyContent: collapsed ? 'center' : 'flex-start',
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            background: token.colorBgContainer,
          }}
        >
          <span
            aria-hidden="true"
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-brand shadow-lg shadow-brand/30"
            style={{ color: textOnBrand }}
          >
            <VideoCameraOutlined style={{ fontSize: 20 }} />
          </span>
          {collapsed ? null : (
            <span style={{ fontWeight: 900, fontSize: 18 }}>
              {(() => {
                const [first, ...rest] = t('common.appName').split(' ');
                return (
                  <>
                    <span style={{ color: token.colorText }}>{first}</span>
                    {rest.length > 0 ? (
                      <span style={{ color: brand.base }}> {rest.join(' ')}</span>
                    ) : null}
                  </>
                );
              })()}
            </span>
          )}
        </div>
        {/* Keep light: the dark antd menu (#001529) is unreadable on a #141414 background. */}
        <div className="p-2">
          {/* Without `inlineCollapsed` the Menu never enters icon-only mode: item labels and group
              titles get clipped by width instead of hiding. */}
          <Menu
            theme="light"
            mode="inline"
            inlineCollapsed={collapsed}
            selectedKeys={current ? [current.path] : []}
            items={menuItems}
          />
        </div>
      </Sider>

      <Layout>
        {/* Header only exposes background/border; real sizing lives in the inner div (index.css reset). */}
        <Header
          className="sticky top-0 z-10 backdrop-blur-md"
          style={{
            // Translucent background per light/dark token (theme switches via antd algorithm, no .dark class).
            background:
              themeMode === 'dark' ? 'rgba(20, 20, 20, 0.75)' : 'rgba(255, 255, 255, 0.8)',
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            boxShadow: token.boxShadowTertiary,
          }}
        >
          <div className="flex h-20 items-center justify-between gap-4 px-6">
            <Breadcrumb
              items={breadcrumbItems}
              separator={<RightOutlined style={{ fontSize: 10 }} />}
            />

            {/* Shared customer dropdown; wrapped in `.cp-customer` so INK classes can read `--cp-ink-rgb`. */}
            <div
              className={themeMode === 'light' ? 'cp-customer cp-customer--light' : 'cp-customer'}
            >
              <AvatarSwitcher variant="operator" />
            </div>
          </div>
        </Header>

        <Content className="m-4">
          <div
            className="min-h-[calc(100vh-160px)] p-5"
            style={{
              background: token.colorBgContainer,
              borderRadius: token.borderRadiusLG,
              boxShadow: token.boxShadowTertiary,
            }}
          >
            <Suspense fallback={<Loading />}>
              <Outlet />
            </Suspense>
          </div>

          {/* Copyright line closing the content frame. */}
          <Footer
            className="text-center"
            style={{ background: 'transparent', color: token.colorTextTertiary, padding: '24px 0' }}
          >
            {t('common.appName')} © {new Date().getFullYear()}
          </Footer>
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
