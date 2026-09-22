import type { PageQuery } from './api';
import type { User } from './auth';

/** Account admin, mirroring internal/dto/user.go. `User` lives in './auth' (authStore uses it). All endpoints here are ADMIN-ONLY (RequireRoles admin on all 4 routes), unlike halls which allow staff. */

export type UserRole = User['role'];

/** Display order; descending privilege for readers, NOT a technical rank: backend matches roles exactly, admin does not imply staff. */
export const USER_ROLES: readonly UserRole[] = ['admin', 'staff', 'customer'] as const;

/** Roles creatable via POST /admin/users (customers self-register; backend binding is oneof=staff admin). */
export const CREATABLE_ROLES: readonly UserRole[] = ['staff', 'admin'] as const;

/** No sort/order: backend always `created_at DESC`. */
export interface UserListQuery extends PageQuery {
  role?: UserRole;
  /** Go pointer: absent means no filter, distinct from false. */
  active?: boolean;
}

/** dto.CreateUserRequest. `role` chi nhan staff hoac admin. */
export interface CreateUserPayload {
  email: string;
  /** min=6, max=72 (bcrypt limit). */
  password: string;
  full_name: string;
  role: UserRole;
}

/** Lock/unlock and role only; cannot edit anyone's name/email/phone. Both fields empty is 400/40001 "nothing to update". */
export interface UpdateUserPayload {
  active?: boolean;
  role?: UserRole;
}
