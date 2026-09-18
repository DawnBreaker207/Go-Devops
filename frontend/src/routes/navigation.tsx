import { useMemo, type ReactNode } from 'react';
import {
  CalendarOutlined,
  DashboardOutlined,
  LayoutOutlined,
  BarChartOutlined,
  ScheduleOutlined,
  TeamOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { useCurrentRole, type UserRole } from '@/hooks/useHasRole';
import { PATHS } from './paths';

/**
 * Man hinh van hanh: backend cho ca admin lan staff doc
 * (RequireRoles(RoleAdmin, RoleStaff) tren /admin/showtimes, /movies writes...).
 */
export const ROLES_OPERATOR: UserRole[] = ['admin', 'staff'];

/**
 * Chi admin. Backend giu rieng cho admin moi view khong gioi han pham vi keo
 * email cua moi khach canh so tien: /admin/orders, /admin/stats, /admin/users,
 * /admin/reports/daily, /admin/audit-logs. Staff doc tung don qua /staff/orders/:id.
 */
export const ROLES_ADMIN: UserRole[] = ['admin'];

export interface NavItem {
  path: string;
  /** menu.<i18nKey> phai ton tai trong CA HAI vi.json va en.json. */
  i18nKey: string;
  icon: ReactNode;
  roles: UserRole[];
}

/**
 * Nguon su that duy nhat cua sider, breadcrumb va quyen vao tung route. Them mot
 * trang top-level = them mot dong o day + mot dong trong routes/index.tsx dung
 * cung hang so roles, khong sua Menu bang tay nua.
 */
export const NAV_ITEMS: NavItem[] = [
  {
    path: PATHS.dashboard,
    i18nKey: 'dashboard',
    icon: <DashboardOutlined />,
    roles: ROLES_OPERATOR,
  },
  { path: PATHS.movies, i18nKey: 'movies', icon: <VideoCameraOutlined />, roles: ROLES_OPERATOR },
  {
    path: PATHS.showtimes,
    i18nKey: 'showtimes',
    icon: <CalendarOutlined />,
    roles: ROLES_OPERATOR,
  },
  { path: PATHS.halls, i18nKey: 'halls', icon: <LayoutOutlined />, roles: ROLES_OPERATOR },
  { path: PATHS.bookings, i18nKey: 'bookings', icon: <ScheduleOutlined />, roles: ROLES_ADMIN },
  { path: PATHS.users, i18nKey: 'users', icon: <TeamOutlined />, roles: ROLES_ADMIN },
  { path: PATHS.reports, i18nKey: 'reports', icon: <BarChartOutlined />, roles: ROLES_ADMIN },
];

/** Cac muc role hien tai thuc su vao duoc. Chua dang nhap thi rong. */
export const useNavItems = (): NavItem[] => {
  const role = useCurrentRole();
  return useMemo(
    () => (role === null ? [] : NAV_ITEMS.filter((item) => item.roles.includes(role))),
    [role]
  );
};

/**
 * Trang dau tien role nay vao duoc sau khi dang nhap.
 *
 * Nguoi van hanh ve muc menu dau tien cua ho; KHACH khong co muc nao nen ve
 * trang chu cua khu khach. Truoc day ham nay tra null cho khach va LoginPage
 * lai mac dinh ve /dashboard, nen khach dang nhap xong roi thang vao man 403 -
 * tuc khong dung duoc app.
 */
export const useLandingPath = (): string => useNavItems()[0]?.path ?? PATHS.home;

/** Muc khop voi URL hien tai, theo tien to duong dan. */
export const findNavItem = (pathname: string): NavItem | undefined =>
  NAV_ITEMS.find((item) => pathname.startsWith(item.path));
