import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { App as AntdApp, ConfigProvider } from 'antd';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import RequireRole from '../RequireRole';
import { ROLES_ADMIN, ROLES_OPERATOR } from '../navigation';
import { useAuthStore } from '@/stores/authStore';
import type { User } from '@/types';

const asRole = (role: User['role']) => {
  useAuthStore.setState({
    user: {
      id: 'u1',
      email: `${role}@test.local`,
      full_name: role,
      role,
      active: true,
      created_at: '2026-01-01T00:00:00+07:00',
      updated_at: '2026-01-01T00:00:00+07:00',
    },
    isAuthenticated: true,
    isBootstrapping: false,
  });
};

const renderGuard = (roles: User['role'][]) => {
  const router = createMemoryRouter(
    [
      {
        element: <RequireRole roles={roles} />,
        children: [{ path: '/bookings', element: <div>noi dung bi khoa</div> }],
      },
    ],
    { initialEntries: ['/bookings'] }
  );
  return render(
    <ConfigProvider>
      <AntdApp>
        <RouterProvider router={router} />
      </AntdApp>
    </ConfigProvider>
  );
};

describe('RequireRole', () => {
  it('cho qua khi role nam trong danh sach', () => {
    asRole('admin');
    renderGuard(ROLES_ADMIN);

    expect(screen.getByText('noi dung bi khoa')).toBeInTheDocument();
  });

  it('chan staff khoi man hinh chi danh cho admin', () => {
    asRole('staff');
    renderGuard(ROLES_ADMIN);

    expect(screen.queryByText('noi dung bi khoa')).not.toBeInTheDocument();
    expect(screen.getByText('403')).toBeInTheDocument();
  });

  it('khong co thu bac: admin khong tu dong la staff', () => {
    asRole('admin');
    renderGuard(['staff']);

    expect(screen.queryByText('noi dung bi khoa')).not.toBeInTheDocument();
    expect(screen.getByText('403')).toBeInTheDocument();
  });

  it('staff vao duoc man hinh van hanh', () => {
    asRole('staff');
    renderGuard(ROLES_OPERATOR);

    expect(screen.getByText('noi dung bi khoa')).toBeInTheDocument();
  });

  it('customers blocked from ops screens still get their own home link', () => {
    // Before the customer zone existed, customers had nowhere to go so the button hid.
    // Now `useLandingPath` returns PATHS.home, which is why LoginPage no longer
    // drops customers straight into /dashboard and a 403.
    asRole('customer');
    renderGuard(ROLES_OPERATOR);

    expect(screen.getByText('403')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Đăng xuất' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Về trang chủ' })).toBeInTheDocument();
  });

  it('con vao duoc trang khac thi co loi ve trang chu', () => {
    asRole('staff');
    renderGuard(ROLES_ADMIN);

    expect(screen.getByRole('button', { name: 'Về trang chủ' })).toBeInTheDocument();
  });
});
