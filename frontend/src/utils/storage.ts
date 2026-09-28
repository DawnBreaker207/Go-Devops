// Storage key prefix: every key starts with cp_.
const ACCESS_TOKEN_KEY = 'cp_access_token';
const REFRESH_TOKEN_KEY = 'cp_refresh_token';

// Read by i18next before any store exists, so it lives here to avoid an import cycle.
export const LANG_STORAGE_KEY = 'cp_language';
export const THEME_STORAGE_KEY = 'cp_theme';
// Separate key on purpose: the two zones toggle independently (operator light, customer dark).
export const CUSTOMER_THEME_STORAGE_KEY = 'cp_customer_theme';

export const ACCESS_TOKEN_STORAGE_KEY = ACCESS_TOKEN_KEY;

export const tokenStorage = {
  getAccessToken: () => localStorage.getItem(ACCESS_TOKEN_KEY),
  getRefreshToken: () => localStorage.getItem(REFRESH_TOKEN_KEY),
  set: (accessToken: string, refreshToken: string) => {
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  },
  clear: () => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  },
};

export const safeStorage = {
  get: <T>(key: string): T | null => {
    try {
      const raw = localStorage.getItem(key);
      return raw ? (JSON.parse(raw) as T) : null;
    } catch {
      return null;
    }
  },
  // Raw read, no JSON parse: legacy values are plain strings ('light'/'dark').
  getRaw: (key: string): string | null => {
    try {
      return localStorage.getItem(key);
    } catch {
      return null;
    }
  },
  set: (key: string, value: unknown) => {
    try {
      localStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value));
    } catch {
      // Ignore: storage is a convenience, never a hard path.
    }
  },
};
