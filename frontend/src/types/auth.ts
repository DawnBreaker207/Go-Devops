/**
 * dto.UserResponse - dung chung cho `GET /users/me`, `POST /auth/login` va
 * `GET /admin/users`. Chi `phone` co omitempty.
 */
export interface User {
  id: string;
  email: string;
  full_name: string;
  phone?: string;
  role: 'admin' | 'staff' | 'customer';
  /** false = tai khoan bi khoa. KHAC voi xoa mem: tai khoan da xoa bien mat han
   *  khoi danh sach vi GORM loc deleted_at, khong the mo khoa lai duoc. */
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
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
