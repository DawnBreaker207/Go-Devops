import { useAuthStore } from '@/stores/authStore';
import type { UserRole } from '@/types';

// Dinh nghia goc nam o '@/types/user'; re-export de cac file dang import tu day
// khong phai doi, va de chi co MOT cho dinh nghia danh sach role.
export type { UserRole };

export const useCurrentRole = (): UserRole | null => useAuthStore((s) => s.user?.role ?? null);

/**
 * Backend so khop role bang map lookup chinh xac, phan biet hoa thuong va KHONG
 * co thu bac (middleware/auth.go RequireRoles): admin khong bao ham staff. Hook
 * nay phai giong het the, nen no chi la mot phep includes, khong co rank.
 *
 * Dung de an menu va khoa nut. Chan ca man hinh thi dung <RequireRole/>.
 */
export const useHasRole = (...roles: UserRole[]): boolean =>
  useAuthStore((s) => (s.user ? roles.includes(s.user.role) : false));
