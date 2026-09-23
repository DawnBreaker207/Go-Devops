import { Suspense } from 'react';
import { Outlet } from 'react-router-dom';
import { Layout, theme as antdTheme } from 'antd';
import Loading from '@/components/Loading';
import { brandAlpha } from '@/theme';

/** Ops-area shell (`/login`): centered antd Card on a light brand-tinted backdrop (no heavy components/logos here). */
export const AuthLayout = () => {
  const { token } = antdTheme.useToken();

  return (
    <Layout
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 24,
        padding: '32px 16px',
        background: token.colorBgLayout,
        backgroundImage: [
          `radial-gradient(560px circle at 12% -10%, ${brandAlpha(0.16)}, transparent 60%)`,
          `radial-gradient(480px circle at 108% 110%, ${brandAlpha(0.12)}, transparent 55%)`,
        ].join(', '),
      }}
    >
      <Suspense fallback={<Loading fullscreen />}>
        <Outlet />
      </Suspense>
    </Layout>
  );
};

export default AuthLayout;
