import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';
import '@/locales/i18n';
import { useAppStore } from '@/stores/appStore';
import { useAuthStore } from '@/stores/authStore';

// Zustand stores are file-wide singletons; snapshot initial state to restore after each test.
const initialAuthState = useAuthStore.getState();
const initialAppState = useAppStore.getState();

afterEach(() => {
  useAuthStore.setState(initialAuthState, true);
  useAppStore.setState(initialAppState, true);
});

// antd reads matchMedia at render time; jsdom has none.
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
