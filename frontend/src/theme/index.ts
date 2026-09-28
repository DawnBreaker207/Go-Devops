import { theme as antdTheme, type ThemeConfig } from 'antd';
import type { ThemeMode } from '@/stores/appStore';
import { brand, brandAlpha, semantic, surface, textOnBrand } from './tokens';

export * from './tokens';

export const buildTheme = (mode: ThemeMode): ThemeConfig => {
  const dark = mode === 'dark';
  const s = dark ? surface.dark : surface.light;

  return {
    algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: brand.base,
      colorInfo: semantic.info,
      colorError: semantic.danger,
      colorWarning: semantic.warning,
      colorSuccess: semantic.success,
      colorTextLightSolid: textOnBrand,
      colorBgLayout: dark ? surface.dark.sunken : surface.light.sunken,
      borderRadius: 6,
      fontSize: 14,
      fontFamily:
        "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    },
    components: {
      Layout: {
        siderBg: dark ? surface.dark.base : surface.light.base,
        headerHeight: 56,
      },
      Menu: {
        itemMarginInline: 8,
        itemHeight: 42,
        itemBorderRadius: 8,
        itemSelectedBg: dark ? brandAlpha(0.16) : brand.soft,
        itemSelectedColor: brand.base,
        itemHoverBg: dark ? brandAlpha(0.1) : brand.softer,
        itemHoverColor: brand.base,
      },
      Table: {
        headerBg: s.sunken,
      },
    },
  };
};
