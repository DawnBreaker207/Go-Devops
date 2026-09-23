import { useMemo, type ReactNode } from 'react';
import {
  CalendarOutlined,
  CoffeeOutlined,
  TagOutlined,
  DashboardOutlined,
  FileSearchOutlined,
  LayoutOutlined,
  BarChartOutlined,
  ScheduleOutlined,
  ShopOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { useCurrentRole, type UserRole } from '@/hooks/useHasRole';
import { PATHS } from './paths';

/** Readable by both admin and staff (backend RequireRoles on /admin/showtimes, /movies writes...). */
export const ROLES_OPERATOR: UserRole[] = ['admin', 'staff'];

/** Admin only: /admin/orders, /stats, /users, /reports/daily, /audit-logs. */
export const ROLES_ADMIN: UserRole[] = ['admin'];

/** Sider display group - one title per group, see menu.group<Name>. */
export type NavGroup = 'overview' | 'operations' | 'management';

export interface NavItem {
  path: string;
  /** menu.<i18nKey> must exist in BOTH vi.json and en.json. */
  i18nKey: string;
  icon: ReactNode;
  roles: UserRole[];
  group: NavGroup;
}

/** Sider, breadcrumb and route access share one source: a new page = 1 line here + 1 router line with the same roles. */
export const NAV_ITEMS: NavItem[] = [
  {
    path: PATHS.dashboard,
    i18nKey: 'dashboard',
    icon: <DashboardOutlined />,
    roles: ROLES_OPERATOR,
    group: 'overview',
  },
  {
    path: PATHS.movies,
    i18nKey: 'movies',
    icon: <VideoCameraOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.showtimes,
    i18nKey: 'showtimes',
    icon: <CalendarOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.halls,
    i18nKey: 'halls',
    icon: <LayoutOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.concessions,
    i18nKey: 'concessions',
    icon: <CoffeeOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.boxOffice,
    i18nKey: 'boxOffice',
    icon: <ShopOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.customerLookup,
    i18nKey: 'customerLookup',
    icon: <TeamOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.bookings,
    i18nKey: 'bookings',
    icon: <ScheduleOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    path: PATHS.users,
    i18nKey: 'users',
    icon: <TeamOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    // ADMIN-ONLY, unlike Bap nuoc next door: a code moves revenue.
    path: PATHS.discounts,
    i18nKey: 'discounts',
    icon: <TagOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    path: PATHS.reports,
    i18nKey: 'reports',
    icon: <BarChartOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    path: PATHS.auditLogs,
    i18nKey: 'auditLogs',
    icon: <FileSearchOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    path: PATHS.batchJobs,
    i18nKey: 'batchJobs',
    icon: <ThunderboltOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
];

/** Empty when signed out. */
export const useNavItems = (): NavItem[] => {
  const role = useCurrentRole();
  return useMemo(
    () => (role === null ? [] : NAV_ITEMS.filter((item) => item.roles.includes(role))),
    [role]
  );
};

/** First page a role can enter (used right after login resolves). Operators land on the first menu item, customers on home. */
export const landingPathForRole = (role: UserRole | null): string =>
  (role === null ? [] : NAV_ITEMS.filter((item) => item.roles.includes(role)))[0]?.path ??
  PATHS.home;

/** Render-only hook version. Never inside async handlers (freezes to render-time role) - call landingPathForRole with the current role there. */
export const useLandingPath = (): string => landingPathForRole(useCurrentRole());

export const findNavItem = (pathname: string): NavItem | undefined =>
  NAV_ITEMS.find((item) => pathname.startsWith(item.path));
