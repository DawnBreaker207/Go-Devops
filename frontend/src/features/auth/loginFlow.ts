import { useAuthStore } from '@/stores/authStore';
import { landingPathForRole } from '@/routes/navigation';
import type { ApiError, LoginRequest } from '@/types';

export type LoginFlowResult = { ok: true; landingPath: string } | { ok: false; message: string };

/** Shared login flow returning the landing path or an error message. */
export async function performLogin(
  payload: LoginRequest,
  fallbackErrorMessage: string
): Promise<LoginFlowResult> {
  try {
    await useAuthStore.getState().login(payload);
    const freshRole = useAuthStore.getState().user?.role ?? null;
    return { ok: true, landingPath: landingPathForRole(freshRole) };
  } catch (error) {
    return { ok: false, message: (error as ApiError).message || fallbackErrorMessage };
  }
}
