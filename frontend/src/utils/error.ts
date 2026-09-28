import type { ApiError } from '@/types';

/** Backend validation code. No 422 anywhere: every validation failure is HTTP 400 with this code and a flat {field: message} details map. */
export const VALIDATION_ERROR_CODE = 40001;

/** normalizeError always throws ApiError, but TS catch is unknown. */
export const isApiError = (error: unknown): error is ApiError =>
  typeof error === 'object' &&
  error !== null &&
  typeof (error as ApiError).code === 'number' &&
  typeof (error as ApiError).message === 'string';

/** Display message; fallback for non-API failures. */
export const errorMessage = (error: unknown, fallback: string): string => {
  if (isApiError(error) && error.message) return error.message;
  return fallback;
};

/** User-safe error text: backend messages leaking internals (SQLSTATE, stack traces) render as the fallback; raw text goes to console for devs. */
const TECHNICAL_PATTERNS = [
  /sqlstate/i,
  /\bERROR:\s/i,
  /on conflict/i,
  /\bpq:/i,
  /\(SQLSTATE/i,
  /at\s+[\w$.]+\s*\(.*:\d+:\d+\)/,
  /null value in column/i,
  /violates .*constraint/i,
];

export const safeMessage = (error: unknown, fallback: string): string => {
  if (!isApiError(error) || !error.message) return fallback;
  if (TECHNICAL_PATTERNS.some((re) => re.test(error.message))) {
    console.error('[api]', error.code, error.message);
    return fallback;
  }
  return error.message;
};

/** details is meaningful only for code 40001. */
export const fieldErrorsOf = (error: unknown): Record<string, string> | undefined => {
  if (!isApiError(error) || error.code !== VALIDATION_ERROR_CODE) return undefined;
  const { details } = error;
  if (!details || Object.keys(details).length === 0) return undefined;
  return details;
};

/** Translated backend reason (err.* namespace) with message fallback. */
export const reasonMessage = (
  error: unknown,
  t: (key: string, options?: { defaultValue?: string }) => string,
  fallback: string
): string => {
  if (isApiError(error) && error.reason) {
    const translated = t(`err.${error.reason}`, { defaultValue: '' });
    if (translated) return translated;
  }
  return safeMessage(error, fallback);
};

const NON_FATAL_REFRESH_REASONS = new Set(['payment_in_progress']);

/** A refresh-flow failure the UI may ride through without logging out. */
export const isRecoverableRefreshError = (error: unknown): boolean =>
  isApiError(error) && !!error.reason && NON_FATAL_REFRESH_REASONS.has(error.reason);
