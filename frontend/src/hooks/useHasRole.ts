import { useAuthStore } from '@/stores/authStore';
import type { UserRole } from '@/types';

// Re-exported so existing imports keep working; the role list has one owner in @/types/user.
export type { UserRole };

export const useCurrentRole = (): UserRole | null => useAuthStore((s) => s.user?.role ?? null);

/** Backend matches roles by exact case-sensitive lookup with no hierarchy (RequireRoles): admin does not imply staff. Plain includes, no rank. Guards menus/buttons; whole screens use <RequireRole/>. */
export const useHasRole = (...roles: UserRole[]): boolean =>
  useAuthStore((s) => (s.user ? roles.includes(s.user.role) : false));
