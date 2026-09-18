import { apiClient, unwrap } from './client';
import type { ApiResponse, LoginRequest, LoginResponse, RegisterRequest, User } from '@/types';

export const authApi = {
  login: (payload: LoginRequest) =>
    apiClient
      .post<ApiResponse<LoginResponse>>('/auth/login', payload, { skipAuthRefresh: true })
      .then(unwrap),

  /** POST /auth/forgot-password - luon 200 du email co ton tai hay khong, de
   *  khong lo ra email nao da dang ky. Link gui qua email tro toi
   *  ACCOUNT_PASSWORD_RESET_URL trong .env cua backend. */
  forgotPassword: (email: string) =>
    apiClient
      .post<ApiResponse<void>>('/auth/forgot-password', { email }, { skipAuthRefresh: true })
      .then(() => undefined),

  /** POST /auth/reset-password - token lay tu link trong email. */
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
