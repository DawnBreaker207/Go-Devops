import { Button, Result, Space } from 'antd';
import { Navigate, Outlet, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Loading from '@/components/Loading';
import { useHasRole, type UserRole } from '@/hooks/useHasRole';
import { useAuthStore } from '@/stores/authStore';
import { useLandingPath } from './navigation';
import { PATHS } from './paths';

interface RequireRoleProps {
  roles: UserRole[];
}

/**
 * Chan theo role o muc route, soi guong dung cach backend chan: RequireRoles la
 * map lookup chinh xac, khong thu bac. Dung lam layout route boc nhom man hinh
 * cung quyen (xem routes/index.tsx).
 *
 * Bao 403 that su chu khong am tham dieu huong: dieu huong ngam se giau mat mot
 * link sai quyen va bien no thanh "app tu nhay lung tung".
 */
export const RequireRole = ({ roles }: RequireRoleProps) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isBootstrapping = useAuthStore((s) => s.isBootstrapping);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const logout = useAuthStore((s) => s.logout);
  const allowed = useHasRole(...roles);
  const landing = useLandingPath();

  // Ton tai de RequireRole dung duoc mot minh; trong router no nam trong
  // ProtectedRoute nen hai nhanh nay gan nhu khong chay.
  if (isBootstrapping) return <Loading fullscreen />;
  if (!isAuthenticated) return <Navigate to={PATHS.login} replace />;

  if (!allowed) {
    return (
      <Result
        status="403"
        title="403"
        subTitle={t('error.forbiddenSubtitle')}
        extra={
          <Space>
            {landing ? (
              <Button type="primary" onClick={() => navigate(landing, { replace: true })}>
                {t('error.backHome')}
              </Button>
            ) : null}
            {/* ProtectedRoute se tu day ve /login ngay khi isAuthenticated tat. */}
            <Button onClick={logout}>{t('common.logout')}</Button>
          </Space>
        }
      />
    );
  }

  return <Outlet />;
};

export default RequireRole;
