import { theme as antdTheme, type ThemeConfig } from 'antd';
import type { ThemeMode } from '@/stores/appStore';
import { brand, brandAlpha, semantic, surface, textOnBrand } from './tokens';

export * from './tokens';

/**
 * Do token cua du an sang token cua antd. Day la cho DUY NHAT antd duoc cau hinh
 * mau; component khong tu viet ma mau.
 */
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
      // Chu tren nen primary. Xem ghi chu textOnBrand trong tokens.ts.
      colorTextLightSolid: textOnBrand,
      colorBgLayout: dark ? surface.dark.sunken : surface.light.sunken,
      borderRadius: 6,
      fontSize: 14,
      fontFamily:
        "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    },
    components: {
      Layout: {
        // Figma de sider mau trang, khong phai navy #001529 mac dinh cua antd.
        siderBg: dark ? surface.dark.base : surface.light.base,
        headerHeight: 56,
      },
      Menu: {
        itemMarginInline: 8,
        // Sider luon chay Menu theme="light": darkAlgorithm da dao bang mau roi.
        // Menu theme="dark" la dien mao navy cu, dat tren nen toi thi chu chim.
        itemSelectedBg: dark ? brandAlpha(0.16) : brand.soft,
        itemSelectedColor: dark ? brand.base : textOnBrand,
      },
      Table: {
        headerBg: s.sunken,
      },
    },
  };
};
