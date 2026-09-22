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
  /** Admin only (staff gets 403). Paginated; fixed created_at DESC sort. */
  list: (query: UserListQuery) =>
    apiClient.get<ApiResponse<PagedData<User>>>('/admin/users', { params: query }).then(unwrap),

  /** Creates staff/admin only; customers self-register via /auth/register. Duplicate email is 409/40900. */
  create: (payload: CreateUserPayload) =>
    apiClient.post<ApiResponse<User>>('/admin/users', payload).then(unwrap),

  /** Lock/unlock and role only. Backend aliases PUT to the same handler but it is undocumented; use PATCH. */
  update: (id: string, payload: UpdateUserPayload) =>
    apiClient.patch<ApiResponse<User>>(`/admin/users/${id}`, payload).then(unwrap),
};
