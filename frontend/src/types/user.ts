import type { PageQuery } from './api';
import type { User } from './auth';

// Mirrors Go DTO internal/dto/user.go.

export type UserRole = User['role'];

export const USER_ROLES: readonly UserRole[] = ['admin', 'staff', 'customer'] as const;

export const CREATABLE_ROLES: readonly UserRole[] = ['staff', 'admin'] as const;

export interface UserListQuery extends PageQuery {
  role?: UserRole;
  active?: boolean;
}

export interface CreateUserPayload {
  email: string;
  password: string;
  full_name: string;
  role: UserRole;
}

export interface UpdateUserPayload {
  active?: boolean;
  role?: UserRole;
}
