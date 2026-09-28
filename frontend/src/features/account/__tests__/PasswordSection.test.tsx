import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/renderWithProviders';
import PasswordSection from '../components/PasswordSection';
import { tokenStorage } from '@/utils/storage';
import { authApi } from '@/api/auth.api';
import type { TokenPair } from '@/types';

/** Full token pair - backend returns token_type and expires_in too. */
const pair = (access: string, refresh: string): TokenPair => ({
  access_token: access,
  refresh_token: refresh,
  token_type: 'Bearer',
  expires_in: 900,
});

// vi.mock can't intercept: authStore already imported the real api; spy on authApi directly instead.
const changePassword = vi.spyOn(authApi, 'changePassword');

const fill = async (
  user: ReturnType<typeof userEvent.setup>,
  current: string,
  next: string,
  confirm = next
) => {
  // Labels must match EXACTLY: "new password" overlaps both fields, a regex would hit the wrong one.
  await user.type(screen.getByLabelText('Mật khẩu hiện tại'), current);
  await user.type(screen.getByLabelText('Mật khẩu mới'), next);
  await user.type(screen.getByLabelText('Nhập lại mật khẩu mới'), confirm);
};

beforeEach(() => {
  changePassword.mockReset();
  tokenStorage.clear();
});

afterEach(() => {
  changePassword.mockReset();
});

describe('PasswordSection', () => {
  // Before this screen, password change only went through the mocked forgot-password email - nearly unchangeable.
  it('gửi mật khẩu hiện tại và mật khẩu mới lên server', async () => {
    changePassword.mockResolvedValue(pair('new-access', 'new-refresh'));
    const user = userEvent.setup();
    renderWithProviders(<PasswordSection />);

    await fill(user, 'cu-rich', 'moi-manh-hon');
    await user.click(screen.getByRole('button', { name: /lưu/i }));

    await waitFor(() => expect(changePassword).toHaveBeenCalledWith('cu-rich', 'moi-manh-hon'));
  });

  // TRAP: changing password revokes every refresh token - skip storing the new pair and this device signs itself out.
  it('lưu cặp token mới, nếu không thiết bị này sẽ tự đăng xuất', async () => {
    tokenStorage.set('token-cu', 'refresh-cu');
    changePassword.mockResolvedValue(pair('token-moi', 'refresh-moi'));
    const user = userEvent.setup();
    renderWithProviders(<PasswordSection />);

    await fill(user, 'cu-rich', 'moi-manh-hon');
    await user.click(screen.getByRole('button', { name: /lưu/i }));

    await waitFor(() => {
      expect(tokenStorage.getAccessToken()).toBe('token-moi');
      expect(tokenStorage.getRefreshToken()).toBe('refresh-moi');
    });
  });

  it('báo thành công và xoá sạch ba ô nhập', async () => {
    changePassword.mockResolvedValue(pair('a', 'b'));
    const user = userEvent.setup();
    renderWithProviders(<PasswordSection />);

    await fill(user, 'cu-rich', 'moi-manh-hon');
    await user.click(screen.getByRole('button', { name: /lưu/i }));

    await waitFor(() => {
      expect(screen.getByLabelText('Mật khẩu hiện tại')).toHaveValue('');
      expect(screen.getByLabelText('Mật khẩu mới')).toHaveValue('');
    });
  });

  // Wrong current password is 401 with no field details, so only a generic message can show.
  it('hiện lỗi khi server từ chối, và không xoá ô nhập', async () => {
    changePassword.mockRejectedValue({ code: 40100, message: 'Mật khẩu hiện tại không đúng.' });
    const user = userEvent.setup();
    renderWithProviders(<PasswordSection />);

    await fill(user, 'sai-roi', 'moi-manh-hon');
    await user.click(screen.getByRole('button', { name: /lưu/i }));

    expect(await screen.findByText(/không đúng/i)).toBeInTheDocument();
    expect(screen.getByLabelText('Mật khẩu hiện tại')).toHaveValue('sai-roi');
  });

  it('không cho gửi khi mật khẩu xác nhận khác mật khẩu mới', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PasswordSection />);

    await fill(user, 'cu-rich', 'moi-manh-hon', 'go-nham');

    expect(screen.getByRole('button', { name: /lưu/i })).toBeDisabled();
    expect(changePassword).not.toHaveBeenCalled();
  });

  it('không cho gửi khi mật khẩu mới ngắn hơn mức backend nhận', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PasswordSection />);

    await fill(user, 'cu-rich', 'abc');

    expect(screen.getByRole('button', { name: /lưu/i })).toBeDisabled();
    expect(changePassword).not.toHaveBeenCalled();
  });
});
