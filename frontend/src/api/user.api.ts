import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  CreateUserPayload,
  PagedData,
  UpdateUserPayload,
  User,
  UserListQuery,
} from '@/types';

export const userApi = {
  list: (query: UserListQuery) =>
    apiClient.get<ApiResponse<PagedData<User>>>('/admin/users', { params: query }).then(unwrap),

  create: (payload: CreateUserPayload) =>
    apiClient.post<ApiResponse<User>>('/admin/users', payload).then(unwrap),

  update: (id: string, payload: UpdateUserPayload) =>
    apiClient.patch<ApiResponse<User>>(`/admin/users/${id}`, payload).then(unwrap),
};
