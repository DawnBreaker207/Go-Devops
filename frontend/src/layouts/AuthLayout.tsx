import { Suspense } from 'react';
import { Outlet } from 'react-router-dom';
import { Layout, theme as antdTheme } from 'antd';
import Loading from '@/components/Loading';

export const AuthLayout = () => {
  const { token } = antdTheme.useToken();

  return (
    <Layout
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: token.colorBgLayout,
      }}
    >
      <Suspense fallback={<Loading fullscreen />}>
        <Outlet />
      </Suspense>
    </Layout>
  );
};

export default AuthLayout;
