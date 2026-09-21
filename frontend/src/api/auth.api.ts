import { apiClient, unwrap } from './client';
import { getDeviceId } from '@/utils/deviceId';
import type { ApiResponse, LoginRequest, LoginResponse, RegisterRequest, User } from '@/types';

export const authApi = {
  /** Attach browser device_id so the session can be listed/revoked via /users/me/sessions. */
  login: (payload: LoginRequest) =>
    apiClient
      .post<ApiResponse<LoginResponse>>(
        '/auth/login',
        { ...payload, device_id: payload.device_id ?? getDeviceId() },
        { skipAuthRefresh: true }
      )
      .then(unwrap),

  /** Always 200 even for unknown email (no enumeration). Link points to backend ACCOUNT_PASSWORD_RESET_URL. */
  forgotPassword: (email: string) =>
    apiClient
      .post<ApiResponse<void>>('/auth/forgot-password', { email }, { skipAuthRefresh: true })
      .then(() => undefined),

  /** Token comes from the emailed link. */
  resetPassword: (token: string, newPassword: string) =>
    apiClient
      .post<ApiResponse<void>>(
        '/auth/reset-password',
        { token, new_password: newPassword },
        { skipAuthRefresh: true }
      )
      .then(() => undefined),

  register: (payload: RegisterRequest) =>
    apiClient
      .post<ApiResponse<User>>('/auth/register', payload, { skipAuthRefresh: true })
      .then(unwrap),

  me: () => apiClient.get<ApiResponse<User>>('/users/me').then(unwrap),
};
