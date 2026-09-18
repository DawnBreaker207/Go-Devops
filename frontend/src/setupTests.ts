import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';
import '@/locales/i18n';
import { useAppStore } from '@/stores/appStore';
import { useAuthStore } from '@/stores/authStore';

// Zustand store la singleton dung chung ca file test. Mot test doi state (hoac
// thay han mot action bang spy) se ro ri sang moi test chay sau no, nen chup lai
// state goc ngay khi nap module va tra ve nguyen ven sau tung test.
const initialAuthState = useAuthStore.getState();
const initialAppState = useAppStore.getState();

afterEach(() => {
  useAuthStore.setState(initialAuthState, true);
  useAppStore.setState(initialAppState, true);
});

// antd doc matchMedia luc render, jsdom chua co san
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }),
});

Object.defineProperty(window, 'getComputedStyle', {
  value: window.getComputedStyle,
});
