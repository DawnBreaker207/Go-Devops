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
  /** GET /admin/users - ADMIN-ONLY (staff nhan 403). Co phan trang.
   *  Tim kiem phu email / ho ten / so dien thoai; sap xep co dinh created_at DESC. */
  list: (query: UserListQuery) =>
    apiClient.get<ApiResponse<PagedData<User>>>('/admin/users', { params: query }).then(unwrap),

  /** POST /admin/users - chi tao duoc tai khoan staff hoac admin; khach hang
   *  tu dang ky qua /auth/register. Trung email la 409/40900. */
  create: (payload: CreateUserPayload) =>
    apiClient.post<ApiResponse<User>>('/admin/users', payload).then(unwrap),

  /** PATCH /admin/users/:id - chi khoa/mo khoa va doi role. Backend co alias PUT
   *  cung handler nhung khong nam trong swagger; dung PATCH. */
  update: (id: string, payload: UpdateUserPayload) =>
    apiClient.patch<ApiResponse<User>>(`/admin/users/${id}`, payload).then(unwrap),
};
