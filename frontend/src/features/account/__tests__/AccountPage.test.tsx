import { describe, expect, it, beforeEach } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/renderWithProviders';
import AccountPage from '../AccountPage';
import { useAuthStore } from '@/stores/authStore';
import type { User } from '@/types';

const user: User = {
  id: 'u1',
  email: 'customer@cinema.local',
  full_name: 'Khach Test',
  phone: '0900000000',
  role: 'customer',
  active: true,
  created_at: '2026-01-01T00:00:00+07:00',
  updated_at: '2026-01-01T00:00:00+07:00',
};

// setupTests restores the store after each test, so setting state directly here is safe.
const signIn = () => useAuthStore.setState({ user, isAuthenticated: true, isBootstrapping: false });

beforeEach(signIn);

describe('AccountPage', () => {
  it('hien du bay muc, dung danh sach dieu huong', async () => {
    renderWithProviders(<AccountPage />, { route: '/account' });

    // Tab list uses tablist/tabs roles, not a <nav>.
    const tablist = screen.getByRole('tablist');
    for (const label of [
      'Hồ sơ',
      'Mật khẩu',
      'Vé của tôi',
      'Giao dịch',
      'Thông báo',
      'Thiết bị',
      'Thành viên',
    ]) {
      expect(within(tablist).getByRole('tab', { name: label })).toBeInTheDocument();
    }
    expect(within(tablist).getAllByRole('tab')).toHaveLength(7);
  });

  it('mac dinh mo muc Ho so', () => {
    renderWithProviders(<AccountPage />, { route: '/account' });
    expect(screen.getByLabelText('Họ và tên')).toBeInTheDocument();
  });

  // Open tab comes from ?tab=, not state - shared links open the right tab, F5 stays put.
  it('mo dung muc tu query string', () => {
    renderWithProviders(<AccountPage />, { route: '/account?tab=password' });
    expect(screen.getByLabelText('Mật khẩu hiện tại')).toBeInTheDocument();
  });

  it('doi muc khi bam, va noi dung doi theo', async () => {
    const u = userEvent.setup();
    renderWithProviders(<AccountPage />, { route: '/account' });

    expect(screen.getByLabelText('Họ và tên')).toBeInTheDocument();

    await u.click(screen.getByRole('tab', { name: 'Mật khẩu' }));

    expect(screen.getByLabelText('Mật khẩu hiện tại')).toBeInTheDocument();
    expect(screen.queryByLabelText('Họ và tên')).not.toBeInTheDocument();
  });

  // An invalid tab in the URL must fall back to Profile, not render an empty frame.
  it('tab la thu khong biet thi ve Ho so', () => {
    renderWithProviders(<AccountPage />, { route: '/account?tab=khong-ton-tai' });
    expect(screen.getByLabelText('Họ và tên')).toBeInTheDocument();
  });

  it('hien email va ten cua nguoi dang dang nhap', () => {
    renderWithProviders(<AccountPage />, { route: '/account' });
    expect(screen.getAllByText('customer@cinema.local').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Khach Test').length).toBeGreaterThan(0);
  });
});
