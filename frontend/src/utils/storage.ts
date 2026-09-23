const ACCESS_TOKEN_KEY = 'cp_access_token';
const REFRESH_TOKEN_KEY = 'cp_refresh_token';

/** Shared by appStore AND locales/i18n (read during i18next init, before any store exists); kept here to avoid an import cycle. */
export const LANG_STORAGE_KEY = 'cp_language';
/** Operator zone (antd algorithm). */
export const THEME_STORAGE_KEY = 'cp_theme';
/** Customer zone, deliberately a SEPARATE key: the two zones have independent switches and defaults
 *  (operator light, customer dark), so one must never overwrite the other. */
export const CUSTOMER_THEME_STORAGE_KEY = 'cp_customer_theme';

/** Access-token key, exported so multi-tab `storage`-event sync doesn't guess the string. */
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

/** Best-effort localStorage (same philosophy as qrCache.ts): every failure (private mode, quota, blocked) is swallowed and reads as absent. Single home for the try/catch previously duplicated per caller. */
export const safeStorage = {
  /** Read + JSON-parse. */
  get: <T>(key: string): T | null => {
    try {
      const raw = localStorage.getItem(key);
      return raw ? (JSON.parse(raw) as T) : null;
    } catch {
      return null;
    }
  },
  /** Raw string read, no JSON parse: preserves the legacy plain-string format (e.g. 'light'/'dark'). */
  getRaw: (key: string): string | null => {
    try {
      return localStorage.getItem(key);
    } catch {
      return null;
    }
  },
  /** Write: raw strings stored as-is, everything else JSON-encoded. */
  set: (key: string, value: unknown) => {
    try {
      localStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value));
    } catch {
      // Ignore: storage is a convenience, never a hard path.
    }
  },
};
