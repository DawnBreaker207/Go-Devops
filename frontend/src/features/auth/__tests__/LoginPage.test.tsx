import { describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { App as AntdApp, ConfigProvider } from 'antd';
import LoginPage from '../LoginPage';
import { useAuthStore } from '@/stores/authStore';

const renderLoginPage = () =>
  render(
    <ConfigProvider>
      <AntdApp>
        <MemoryRouter>
          <LoginPage />
        </MemoryRouter>
      </AntdApp>
    </ConfigProvider>
  );

describe('LoginPage', () => {
  it('hien thi form dang nhap', () => {
    renderLoginPage();

    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Mật khẩu')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Đăng nhập' })).toBeInTheDocument();
  });

  it('bao loi validate khi submit form rong', async () => {
    const user = userEvent.setup();
    renderLoginPage();

    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }));

    await waitFor(() => {
      expect(screen.getByText('Vui lòng nhập email')).toBeInTheDocument();
      expect(screen.getByText('Vui lòng nhập mật khẩu')).toBeInTheDocument();
    });
  });

  it('goi login store voi thong tin da nhap', async () => {
    const loginSpy = vi.fn().mockResolvedValue(undefined);
    useAuthStore.setState({ login: loginSpy });

    const user = userEvent.setup();
    renderLoginPage();

    await user.type(screen.getByLabelText('Email'), 'admin@cinema.local');
    await user.type(screen.getByLabelText('Mật khẩu'), 'secret123');
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }));

    await waitFor(() => {
      expect(loginSpy).toHaveBeenCalledWith({
        email: 'admin@cinema.local',
        password: 'secret123',
      });
    });
  });
});
