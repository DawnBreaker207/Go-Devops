import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import Loading from '@/components/Loading';
import { PATHS } from './paths';

export const ProtectedRoute = () => {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isBootstrapping = useAuthStore((s) => s.isBootstrapping);
  const location = useLocation();

  if (isBootstrapping) return <Loading fullscreen />;

  if (!isAuthenticated) {
    return <Navigate to={PATHS.login} replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
};

export default ProtectedRoute;
