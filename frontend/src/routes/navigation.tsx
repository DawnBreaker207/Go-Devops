import { useMemo, type ReactNode } from 'react';
import {
  DashboardOutlined,
  DollarOutlined,
  FileSearchOutlined,
  GiftOutlined,
  ScheduleOutlined,
  ShopOutlined,
  TeamOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { useCurrentRole, type UserRole } from '@/hooks/useHasRole';
import { PATHS } from './paths';

export const ROLES_OPERATOR: UserRole[] = ['admin', 'staff'];

export const ROLES_ADMIN: UserRole[] = ['admin'];

export type NavGroup = 'overview' | 'operations' | 'management';

export interface NavItem {
  path: string;
  i18nKey: string;
  icon: ReactNode;
  roles: UserRole[];
  group: NavGroup;
}

export const NAV_ITEMS: NavItem[] = [
  {
    path: PATHS.dashboard,
    i18nKey: 'dashboard',
    icon: <DashboardOutlined />,
    roles: ROLES_OPERATOR,
    group: 'overview',
  },
  {
    path: PATHS.catalog,
    i18nKey: 'catalog',
    icon: <VideoCameraOutlined />,
    roles: ROLES_OPERATOR,
    group: 'operations',
  },
  {
    path: PATHS.counter,
    i18nKey: 'counter',
    icon: <ShopOutlined />,
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
    path: PATHS.promotions,
    i18nKey: 'promotions',
    icon: <GiftOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    path: PATHS.pricingAdmin,
    i18nKey: 'pricingAdmin',
    icon: <DollarOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
  {
    path: PATHS.monitoring,
    i18nKey: 'monitoring',
    icon: <FileSearchOutlined />,
    roles: ROLES_ADMIN,
    group: 'management',
  },
];

export const useNavItems = (): NavItem[] => {
  const role = useCurrentRole();
  return useMemo(
    () => (role === null ? [] : NAV_ITEMS.filter((item) => item.roles.includes(role))),
    [role]
  );
};

export const landingPathForRole = (role: UserRole | null): string =>
  (role === null ? [] : NAV_ITEMS.filter((item) => item.roles.includes(role)))[0]?.path ??
  PATHS.home;

export const useLandingPath = (): string => landingPathForRole(useCurrentRole());

export const findNavItem = (pathname: string): NavItem | undefined =>
  NAV_ITEMS.find((item) => pathname.startsWith(item.path));
