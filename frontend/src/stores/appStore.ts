import { create } from 'zustand';
import i18n, { type AppLanguage } from '@/locales/i18n';

export type ThemeMode = 'light' | 'dark';

const THEME_KEY = 'cp_theme';
const LANG_KEY = 'cp_language';

interface AppState {
  theme: ThemeMode;
  language: AppLanguage;
  siderCollapsed: boolean;
  toggleTheme: () => void;
  setLanguage: (lang: AppLanguage) => void;
  toggleSider: () => void;
}

const initialTheme = (localStorage.getItem(THEME_KEY) as ThemeMode | null) ?? 'light';
const initialLanguage = (localStorage.getItem(LANG_KEY) as AppLanguage | null) ?? 'vi';

export const useAppStore = create<AppState>((set, get) => ({
  theme: initialTheme,
  language: initialLanguage,
  siderCollapsed: false,

  toggleTheme: () => {
    const next: ThemeMode = get().theme === 'light' ? 'dark' : 'light';
    localStorage.setItem(THEME_KEY, next);
    set({ theme: next });
  },

  setLanguage: (lang) => {
    localStorage.setItem(LANG_KEY, lang);
    void i18n.changeLanguage(lang);
    set({ language: lang });
  },

  toggleSider: () => set((state) => ({ siderCollapsed: !state.siderCollapsed })),
}));
