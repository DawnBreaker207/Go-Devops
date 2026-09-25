import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';
import type { ApiError, ApiResponse, TokenPair } from '@/types';
import { tokenStorage } from '@/utils/storage';

/** Fired on expired session; App listens and routes to /login. */
export const UNAUTHORIZED_EVENT = 'cp:unauthorized';

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

interface RetriableConfig extends InternalAxiosRequestConfig {
  _retry?: boolean;
  skipAuthRefresh?: boolean;
}

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30_000,
  headers: { 'Content-Type': 'application/json' },
});

/** Separate client so refresh never re-enters the interceptor loop. */
const refreshClient = axios.create({ baseURL: API_BASE_URL, timeout: 30_000 });

apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = tokenStorage.getAccessToken();
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

let isRefreshing = false;
let pendingQueue: Array<(token: string | null) => void> = [];

const flushQueue = (token: string | null) => {
  pendingQueue.forEach((resolve) => resolve(token));
  pendingQueue = [];
};

const forceLogout = () => {
  tokenStorage.clear();
  window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT));
};

const normalizeError = (error: AxiosError<ApiError>): ApiError => {
  if (error.response?.data && typeof error.response.data === 'object') {
    const data = error.response.data;
    return {
      code: data.code ?? error.response.status,
      message: data.message ?? error.message,
      details: data.details,
    };
  }
  if (error.code === 'ECONNABORTED') {
    return { code: 408, message: 'Request timeout' };
  }
  return { code: error.response?.status ?? 0, message: error.message || 'Network error' };
};

apiClient.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError<ApiError>) => {
    const config = error.config as RetriableConfig | undefined;
    const status = error.response?.status;

    if (status !== 401 || !config || config._retry || config.skipAuthRefresh) {
      return Promise.reject(normalizeError(error));
    }

    const refreshToken = tokenStorage.getRefreshToken();
    if (!refreshToken) {
      forceLogout();
      return Promise.reject(normalizeError(error));
    }

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        pendingQueue.push((token) => {
          if (!token) {
            reject(normalizeError(error));
            return;
          }
          config._retry = true;
          config.headers.set('Authorization', `Bearer ${token}`);
          resolve(apiClient(config));
        });
      });
    }

    isRefreshing = true;
    config._retry = true;

    try {
      const { data } = await refreshClient.post<ApiResponse<TokenPair>>('/auth/refresh', {
        refresh_token: refreshToken,
      });
      const pair = data.data;
      tokenStorage.set(pair.access_token, pair.refresh_token);
      flushQueue(pair.access_token);
      config.headers.set('Authorization', `Bearer ${pair.access_token}`);
      return await apiClient(config);
    } catch (refreshError) {
      flushQueue(null);
      forceLogout();
      return await Promise.reject(normalizeError(refreshError as AxiosError<ApiError>));
    } finally {
      isRefreshing = false;
    }
  }
);

/** Unwrap { code, message, data } to the payload. */
export const unwrap = <T>(response: AxiosResponse<ApiResponse<T>>): T => response.data.data;

declare module 'axios' {
  export interface AxiosRequestConfig {
    /** Skip auto refresh for this request. */
    skipAuthRefresh?: boolean;
  }
}
