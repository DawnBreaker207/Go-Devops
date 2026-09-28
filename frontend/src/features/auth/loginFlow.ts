import type { TFunction } from 'i18next';
import { useAuthStore } from '@/stores/authStore';
import { landingPathForRole } from '@/routes/navigation';
import type { LoginRequest } from '@/types';
import { fieldErrorsOf, reasonMessage } from '@/utils/error';

export type LoginFlowResult =
  | { ok: true; landingPath: string }
  // Prefer 400/40001 fieldErrors over the generic message.
  | { ok: false; message: string; fieldErrors?: Record<string, string> };

export async function performLogin(payload: LoginRequest, t: TFunction): Promise<LoginFlowResult> {
  try {
    await useAuthStore.getState().login(payload);
    const freshRole = useAuthStore.getState().user?.role ?? null;
    return { ok: true, landingPath: landingPathForRole(freshRole) };
  } catch (error) {
    return {
      ok: false,
      message: reasonMessage(error, t, t('auth.loginFailed')),
      fieldErrors: fieldErrorsOf(error),
    };
  }
}
