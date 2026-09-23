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
