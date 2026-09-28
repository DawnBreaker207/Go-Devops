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

// Exact role match: no hierarchy, wrong role returns 403.
export const RequireRole = ({ roles }: RequireRoleProps) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isBootstrapping = useAuthStore((s) => s.isBootstrapping);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const logout = useAuthStore((s) => s.logout);
  const allowed = useHasRole(...roles);
  const landing = useLandingPath();

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
            <Button type="primary" onClick={() => navigate(landing, { replace: true })}>
              {t('error.backHome')}
            </Button>
            <Button onClick={logout}>{t('common.logout')}</Button>
          </Space>
        }
      />
    );
  }

  return <Outlet />;
};

export default RequireRole;
