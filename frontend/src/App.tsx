import { useEffect, useState } from 'react';
import { RouterProvider } from 'react-router-dom';
import { App as AntdApp, ConfigProvider } from 'antd';
import enUS from 'antd/locale/en_US';
import viVN from 'antd/locale/vi_VN';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import 'dayjs/locale/vi';
import { router } from '@/routes';
import { PATHS } from '@/routes/paths';
import { UNAUTHORIZED_EVENT } from '@/api/client';
import { useAppStore } from '@/stores/appStore';
import { useAuthStore } from '@/stores/authStore';
import { buildTheme } from '@/theme';
import ErrorBoundary from '@/components/ErrorBoundary';

const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: 1,
        refetchOnWindowFocus: false,
        staleTime: 30_000,
      },
    },
  });

export const App = () => {
  const [queryClient] = useState(createQueryClient);
  const themeMode = useAppStore((s) => s.theme);
  const language = useAppStore((s) => s.language);
  const bootstrap = useAuthStore((s) => s.bootstrap);
  const logout = useAuthStore((s) => s.logout);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  // Interceptor ban event nay khi refresh token that bai -> dua ve trang login
  useEffect(() => {
    const handleUnauthorized = () => {
      logout();
      queryClient.clear();
      void router.navigate(PATHS.login, { replace: true });
    };
    window.addEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
  }, [logout, queryClient]);

  return (
    <ConfigProvider theme={buildTheme(themeMode)} locale={language === 'vi' ? viVN : enUS}>
      <AntdApp>
        <ErrorBoundary>
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </ErrorBoundary>
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
