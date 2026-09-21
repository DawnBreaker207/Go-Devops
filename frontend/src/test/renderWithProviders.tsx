import type { ReactElement, ReactNode } from 'react';
import { render, type RenderOptions, type RenderResult } from '@testing-library/react';
import { App as AntdApp, ConfigProvider } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

/** Test providers mirroring the app with isolated routing and no query retry. */
export const createTestQueryClient = (): QueryClient =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });

interface ProvidersProps {
  children: ReactNode;
  route?: string;
  queryClient?: QueryClient;
}

export const Providers = ({ children, route = '/', queryClient }: ProvidersProps) => (
  <ConfigProvider>
    <AntdApp>
      <QueryClientProvider client={queryClient ?? createTestQueryClient()}>
        <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
      </QueryClientProvider>
    </AntdApp>
  </ConfigProvider>
);

interface RenderWithProvidersOptions extends Omit<RenderOptions, 'wrapper'> {
  route?: string;
  queryClient?: QueryClient;
}

export const renderWithProviders = (
  ui: ReactElement,
  { route, queryClient, ...options }: RenderWithProvidersOptions = {}
): RenderResult =>
  render(ui, {
    wrapper: ({ children }) => (
      <Providers route={route} queryClient={queryClient}>
        {children}
      </Providers>
    ),
    ...options,
  });
