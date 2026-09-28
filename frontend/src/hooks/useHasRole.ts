import { useAuthStore } from '@/stores/authStore';
import type { UserRole } from '@/types';

export type { UserRole };

export const useCurrentRole = (): UserRole | null => useAuthStore((s) => s.user?.role ?? null);

// Exact role match: no hierarchy, admin does not imply staff.
export const useHasRole = (...roles: UserRole[]): boolean =>
  useAuthStore((s) => (s.user ? roles.includes(s.user.role) : false));
