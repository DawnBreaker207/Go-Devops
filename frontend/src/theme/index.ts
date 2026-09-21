import { theme as antdTheme, type ThemeConfig } from 'antd';
import type { ThemeMode } from '@/stores/appStore';
import { brand, brandAlpha, semantic, surface, textOnBrand } from './tokens';

export * from './tokens';

/** Map project tokens onto antd. Only place where antd colors are configured. */
export const buildTheme = (mode: ThemeMode): ThemeConfig => {
  const dark = mode === 'dark';
  const s = dark ? surface.dark : surface.light;

  return {
    algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: brand.base,
      colorInfo: brand.base,
      colorError: semantic.danger,
      colorWarning: semantic.warning,
      colorSuccess: semantic.success,
      // Text on primary. See textOnBrand in tokens.ts.
      colorTextLightSolid: textOnBrand,
      colorBgLayout: dark ? surface.dark.sunken : surface.light.sunken,
      borderRadius: 6,
      fontSize: 14,
      fontFamily:
        "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    },
    components: {
      Layout: {
        // Sider is white per Figma, not antd's default navy #001529.
        siderBg: dark ? surface.dark.base : surface.light.base,
        headerHeight: 56,
      },
      Menu: {
        itemMarginInline: 8,
        itemHeight: 42,
        itemBorderRadius: 8,
        // Sider always runs Menu theme="light": darkAlgorithm already inverts the palette.
        itemSelectedBg: dark ? brandAlpha(0.16) : brand.soft,
        // White on brand.soft is unreadable; selected item uses brand.base in both modes.
        itemSelectedColor: brand.base,
        // No antd hover token exists; declare explicitly instead of gray default.
        itemHoverBg: dark ? brandAlpha(0.1) : brand.softer,
        itemHoverColor: brand.base,
      },
      Table: {
        headerBg: s.sunken,
      },
    },
  };
};
