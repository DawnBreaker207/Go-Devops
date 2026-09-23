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

  /** Revokes the WHOLE refresh-token family, so every device on that family is
   *  signed out server-side. `skipAuthRefresh` because a 401 here must not kick
   *  the interceptor into refreshing a token we are in the middle of revoking. */
  logout: (refreshToken: string) =>
    apiClient
      .post<ApiResponse<void>>(
        '/auth/logout',
        { refresh_token: refreshToken },
        { skipAuthRefresh: true }
      )
      .then(() => undefined),

  /** Returns a FRESH token pair, and the caller MUST store it.
   *
   *  The backend revokes every refresh token the user has (RevokeUser) before
   *  issuing this pair, so keeping the old one leaves the session alive only
   *  until the current access token expires and then dies at the first refresh.
   *  Signing the user out of their other devices is the intended behaviour. */
  changePassword: (currentPassword: string, newPassword: string) =>
    apiClient
      .put<ApiResponse<TokenPair>>('/users/me/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      .then(unwrap),

  me: () => apiClient.get<ApiResponse<User>>('/users/me').then(unwrap),
};
