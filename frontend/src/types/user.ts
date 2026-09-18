import type { PageQuery } from './api';
import type { User } from './auth';

/**
 * Quan tri tai khoan - mirror cua internal/dto/user.go. Ban than `User` (tuc
 * dto.UserResponse) nam o './auth' vi authStore cung dung no.
 *
 * Ca nhom endpoint nay la ADMIN-ONLY, khong phai admin+staff nhu nhom phong
 * chieu: `RequireRoles(models.RoleAdmin)` tren ca 4 route.
 */

export type UserRole = User['role'];

/** Thu tu hien thi; cung la thu tu quyen giam dan cho nguoi doc, KHONG phai
 *  thu bac ky thuat - backend so khop role chinh xac, admin khong bao ham staff. */
export const USER_ROLES: readonly UserRole[] = ['admin', 'staff', 'customer'] as const;

/**
 * Role duoc phep TAO qua `POST /admin/users`. Khach hang tu dang ky lay,
 * nen binding cua backend chi nhan `oneof=staff admin`.
 */
export const CREATABLE_ROLES: readonly UserRole[] = ['staff', 'admin'] as const;

/** dto.UserListQuery. Khong co sort/order - backend luon `created_at DESC`. */
export interface UserListQuery extends PageQuery {
  role?: UserRole;
  /** Con tro ben Go: bo trong la khong loc, khac han voi false. */
  active?: boolean;
}

/** dto.CreateUserRequest. `role` chi nhan staff hoac admin. */
export interface CreateUserPayload {
  email: string;
  /** min=6, max=72 (tran cua bcrypt). */
  password: string;
  full_name: string;
  role: UserRole;
}

/**
 * dto.UpdateUserRequest - `PATCH /admin/users/:id` (va alias `PUT` khong co
 * trong swagger). Chi khoa/mo khoa va doi role; khong sua duoc ten, email hay
 * so dien thoai cua nguoi khac.
 *
 * Gui ca hai field deu trong la 400/40001 "nothing to update".
 */
export interface UpdateUserPayload {
  active?: boolean;
  role?: UserRole;
}
