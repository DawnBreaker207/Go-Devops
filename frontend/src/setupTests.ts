import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';
import '@/locales/i18n';

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
