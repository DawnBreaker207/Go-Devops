import { useEffect, useState } from 'react';
import { RouterProvider } from 'react-router-dom';
import { App as AntdApp, ConfigProvider, type ConfigProviderProps } from 'antd';
import enUS from 'antd/locale/en_US';
import viVN from 'antd/locale/vi_VN';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import dayjs from 'dayjs';
import 'dayjs/locale/vi';
import 'dayjs/locale/en';
import { router } from '@/routes';
import { PATHS } from '@/routes/paths';
import { UNAUTHORIZED_EVENT } from '@/api/client';
import { useAppStore } from '@/stores/appStore';
import { useAuthStore } from '@/stores/authStore';
import { buildTheme } from '@/theme';
import ErrorBoundary from '@/components/ErrorBoundary';

type AntdLocale = ConfigProviderProps['locale'];

// Unwrap the double-defaulted antd locale, else it silently falls back to English.
const unwrapLocale = (mod: unknown): AntdLocale => {
  const once = (mod as { default?: unknown }).default ?? mod;
  return ((once as { default?: unknown }).default ?? once) as AntdLocale;
};

const ANTD_LOCALE: Record<'vi' | 'en', AntdLocale> = {
  vi: unwrapLocale(viVN),
  en: unwrapLocale(enUS),
};

const OPERATOR_ONLY_PATHS = [
  PATHS.login,
  PATHS.dashboard,
  PATHS.profile,
  PATHS.catalog,
  PATHS.counter,
  PATHS.bookings,
  PATHS.users,
  PATHS.promotions,
  PATHS.pricingAdmin,
  PATHS.monitoring,
];

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

  useEffect(() => {
    dayjs.locale(language);
  }, [language]);

  useEffect(() => {
    const handleUnauthorized = () => {
      logout();
      queryClient.clear();
      const path = window.location.pathname;
      const isOperatorArea = OPERATOR_ONLY_PATHS.some(
        (p) => path === p || path.startsWith(`${p}/`)
      );
      if (isOperatorArea) {
        void router.navigate(PATHS.login, { replace: true });
      }
    };
    window.addEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, handleUnauthorized);
  }, [logout, queryClient]);

  return (
    <ConfigProvider theme={buildTheme(themeMode)} locale={ANTD_LOCALE[language]}>
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
