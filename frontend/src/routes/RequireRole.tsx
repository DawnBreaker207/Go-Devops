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

/** Route-level role gate: exact-map lookup, no hierarchy. Reports a real 403 instead of silently redirecting (silent redirects disguise wrong-role links as random jumps). */
export const RequireRole = ({ roles }: RequireRoleProps) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isBootstrapping = useAuthStore((s) => s.isBootstrapping);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const logout = useAuthStore((s) => s.logout);
  const allowed = useHasRole(...roles);
  const landing = useLandingPath();

  // Exists so RequireRole works standalone; inside the router it nests under
  // ProtectedRoute, so these branches barely run.
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
            {/* landing always resolves: role-less customers fall back to the customer home. */}
            <Button type="primary" onClick={() => navigate(landing, { replace: true })}>
              {t('error.backHome')}
            </Button>
            {/* ProtectedRoute pushes back to /login the moment auth drops. */}
            <Button onClick={logout}>{t('common.logout')}</Button>
          </Space>
        }
      />
    );
  }

  return <Outlet />;
};

export default RequireRole;
