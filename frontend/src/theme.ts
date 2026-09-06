import { theme as antdTheme, type ThemeConfig } from 'antd';
import type { ThemeMode } from '@/stores/appStore';

/** Design token dung chung, doi mau chu dao tai day */
const baseToken: ThemeConfig['token'] = {
  colorPrimary: '#e50914',
  colorInfo: '#e50914',
  borderRadius: 6,
  fontSize: 14,
  fontFamily:
    "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
};

export const buildTheme = (mode: ThemeMode): ThemeConfig => ({
  algorithm: mode === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
  token: {
    ...baseToken,
    colorBgLayout: mode === 'dark' ? '#141414' : '#f5f5f5',
  },
  components: {
    Layout: {
      siderBg: '#001529',
      headerHeight: 56,
    },
    Menu: {
      itemMarginInline: 8,
    },
    Table: {
      headerBg: mode === 'dark' ? '#1f1f1f' : '#fafafa',
    },
  },
});
