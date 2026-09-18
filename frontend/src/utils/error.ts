import type { ApiError } from '@/types';

/**
 * Ma loi validate cua BackEnd-CP. Khong co 422 o dau trong backend: moi loi
 * validate deu la HTTP 400 kem code nay va mot map phang details {field: message}.
 */
export const VALIDATION_ERROR_CODE = 40001;

/** apiClient.normalizeError luon nem ra ApiError, nhung catch cua TS la unknown. */
export const isApiError = (error: unknown): error is ApiError =>
  typeof error === 'object' &&
  error !== null &&
  typeof (error as ApiError).code === 'number' &&
  typeof (error as ApiError).message === 'string';

/** Lay message de hien thi; fallback dung khi loi khong den tu API. */
export const errorMessage = (error: unknown, fallback: string): string => {
  if (isApiError(error) && error.message) return error.message;
  return fallback;
};

/** details chi co y nghia khi code dung la 40001. */
export const fieldErrorsOf = (error: unknown): Record<string, string> | undefined => {
  if (!isApiError(error) || error.code !== VALIDATION_ERROR_CODE) return undefined;
  const { details } = error;
  if (!details || Object.keys(details).length === 0) return undefined;
  return details;
};
