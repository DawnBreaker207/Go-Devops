import { apiClient, unwrap } from './client';
import type { ApiResponse, LoginRequest, LoginResponse, RegisterRequest, User } from '@/types';

export const authApi = {
  login: (payload: LoginRequest) =>
    apiClient
      .post<ApiResponse<LoginResponse>>('/auth/login', payload, { skipAuthRefresh: true })
      .then(unwrap),

  register: (payload: RegisterRequest) =>
    apiClient
      .post<ApiResponse<User>>('/auth/register', payload, { skipAuthRefresh: true })
      .then(unwrap),

  me: () => apiClient.get<ApiResponse<User>>('/users/me').then(unwrap),
};
