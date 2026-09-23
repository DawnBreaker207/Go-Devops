import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import vi from './vi.json';
import en from './en.json';

export const SUPPORTED_LANGUAGES = ['vi', 'en'] as const;
export type AppLanguage = (typeof SUPPORTED_LANGUAGES)[number];

const stored = localStorage.getItem('cp_language') as AppLanguage | null;

void i18n.use(initReactI18next).init({
  resources: {
    vi: { translation: vi },
    en: { translation: en },
  },
  lng: stored ?? 'vi',
  fallbackLng: 'vi',
  interpolation: { escapeValue: false },
});

export default i18n;
