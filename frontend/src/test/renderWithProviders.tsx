import type { ReactElement, ReactNode } from 'react';
import { render, type RenderOptions, type RenderResult } from '@testing-library/react';
import { App as AntdApp, ConfigProvider } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

/**
 * Cung chuoi provider voi App.tsx, tru RouterProvider: test dung MemoryRouter de
 * chon san URL. Man hinh nao tu dung data router thi tu tao createMemoryRouter.
 *
 * QueryClient tao moi cho tung lan render va TAT retry - retry:1 cua app se lam
 * mot test loi phai cho het lan thu hai roi moi fail.
 */
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
