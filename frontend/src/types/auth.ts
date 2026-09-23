/** dto.UserResponse, shared by GET /users/me, POST /auth/login, GET /admin/users. Only `phone` is omitempty. */
export interface User {
  id: string;
  email: string;
  full_name: string;
  phone?: string;
  role: 'admin' | 'staff' | 'customer';
  /** false = locked. Unlike soft-delete: deleted accounts vanish from lists (GORM filters deleted_at) and can't be unlocked. */
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
  /** Backend omitempty (max=255). Auto-attached in authApi.login; never pass by hand. */
  device_id?: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  full_name: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface LoginResponse extends TokenPair {
  user: User;
}
