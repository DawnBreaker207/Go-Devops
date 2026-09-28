import { apiClient, unwrap } from './client';
import { getDeviceId } from '@/utils/deviceId';
import type {
  ApiResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  TokenPair,
  User,
} from '@/types';

export const authApi = {
  login: (payload: LoginRequest) =>
    apiClient
      .post<ApiResponse<LoginResponse>>(
        '/auth/login',
        { ...payload, device_id: payload.device_id ?? getDeviceId() },
        { skipAuthRefresh: true }
      )
      .then(unwrap),

  forgotPassword: (email: string) =>
    apiClient
      .post<ApiResponse<void>>('/auth/forgot-password', { email }, { skipAuthRefresh: true })
      .then(() => undefined),

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

  logout: (refreshToken: string) =>
    apiClient
      .post<ApiResponse<void>>(
        '/auth/logout',
        { refresh_token: refreshToken },
        { skipAuthRefresh: true }
      )
      .then(() => undefined),

  changePassword: (currentPassword: string, newPassword: string) =>
    apiClient
      .put<ApiResponse<TokenPair>>('/users/me/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      .then(unwrap),

  me: () => apiClient.get<ApiResponse<User>>('/users/me').then(unwrap),
};
