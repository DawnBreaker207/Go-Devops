import { create } from 'zustand';
import i18n, { type AppLanguage } from '@/locales/i18n';
import { LANG_STORAGE_KEY, safeStorage, THEME_STORAGE_KEY } from '@/utils/storage';

export type ThemeMode = 'light' | 'dark';

interface AppState {
  theme: ThemeMode;
  language: AppLanguage;
  siderCollapsed: boolean;
  toggleTheme: () => void;
  setLanguage: (lang: AppLanguage) => void;
  toggleSider: () => void;
}

const initialTheme = (safeStorage.getRaw(THEME_STORAGE_KEY) as ThemeMode | null) ?? 'light';
const initialLanguage = (safeStorage.getRaw(LANG_STORAGE_KEY) as AppLanguage | null) ?? 'vi';

export const useAppStore = create<AppState>((set, get) => ({
  theme: initialTheme,
  language: initialLanguage,
  siderCollapsed: false,

  toggleTheme: () => {
    const next: ThemeMode = get().theme === 'light' ? 'dark' : 'light';
    safeStorage.set(THEME_STORAGE_KEY, next);
    set({ theme: next });
  },

  setLanguage: (lang) => {
    safeStorage.set(LANG_STORAGE_KEY, lang);
    void i18n.changeLanguage(lang);
    set({ language: lang });
  },

  toggleSider: () => set((state) => ({ siderCollapsed: !state.siderCollapsed })),
}));
