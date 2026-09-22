import { create } from 'zustand';
import i18n, { type AppLanguage } from '@/locales/i18n';
import {
  CUSTOMER_THEME_STORAGE_KEY,
  LANG_STORAGE_KEY,
  safeStorage,
  THEME_STORAGE_KEY,
} from '@/utils/storage';

export type ThemeMode = 'light' | 'dark';

/** TWO independent themes, one per zone - they are not the same switch and must never share a value.
 *  `theme` drives antd's algorithm for the operator screens (light by default: the Figma admin is white).
 *  `customerTheme` drives `.cp-customer--light` on CustomerLayout (DARK by default: the customer zone is a
 *  cinema, its backdrop is `cinemaGradient` derived from brand.base, and index.css treats light as opt-in).
 *  Collapsing them into one value turns the admin tables dark the moment a customer picks a night theme. */
interface AppState {
  theme: ThemeMode;
  customerTheme: ThemeMode;
  language: AppLanguage;
  siderCollapsed: boolean;
  toggleTheme: () => void;
  toggleCustomerTheme: () => void;
  setLanguage: (lang: AppLanguage) => void;
  toggleSider: () => void;
}

const initialTheme = (safeStorage.getRaw(THEME_STORAGE_KEY) as ThemeMode | null) ?? 'light';
const initialCustomerTheme =
  (safeStorage.getRaw(CUSTOMER_THEME_STORAGE_KEY) as ThemeMode | null) ?? 'dark';
const initialLanguage = (safeStorage.getRaw(LANG_STORAGE_KEY) as AppLanguage | null) ?? 'vi';

export const useAppStore = create<AppState>((set, get) => ({
  theme: initialTheme,
  customerTheme: initialCustomerTheme,
  language: initialLanguage,
  siderCollapsed: false,

  toggleTheme: () => {
    const next: ThemeMode = get().theme === 'light' ? 'dark' : 'light';
    safeStorage.set(THEME_STORAGE_KEY, next);
    set({ theme: next });
  },

  toggleCustomerTheme: () => {
    const next: ThemeMode = get().customerTheme === 'light' ? 'dark' : 'light';
    safeStorage.set(CUSTOMER_THEME_STORAGE_KEY, next);
    set({ customerTheme: next });
  },

  setLanguage: (lang) => {
    safeStorage.set(LANG_STORAGE_KEY, lang);
    void i18n.changeLanguage(lang);
    set({ language: lang });
  },

  toggleSider: () => set((state) => ({ siderCollapsed: !state.siderCollapsed })),
}));
