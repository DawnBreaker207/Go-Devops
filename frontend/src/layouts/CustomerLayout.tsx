import { Suspense } from 'react';
import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Loading from '@/components/Loading';
import { useAuthStore } from '@/stores/authStore';
import { useHasRole } from '@/hooks/useHasRole';
import { cinemaGradient } from '@/theme';
import { PATHS } from '@/routes/paths';
import './CustomerLayout.css';

/**
 * Vo cua khu khach hang: thanh dau + noi dung, KHONG co sider.
 *
 * Day la mot the gioi thi giac khac han `MainLayout`: nen toi nga xanh, chu
 * trang, va khong dung component nang cua antd (xem `.claude/context/decisions.md`
 * #15). Dark mode cua antd khong cham toi day.
 */
export const CustomerLayout = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const logout = useAuthStore((s) => s.logout);
  // Chi KHACH moi co ve de xem. Admin/staff dang nhap vao day thi moi
  // /orders/* deu tra 403, nen khong mo link "Ve cua toi" cho ho.
  const isCustomer = useHasRole('customer');

  const handleLogout = () => {
    logout();
    navigate(PATHS.home, { replace: true });
  };

  return (
    <div className="cp-customer" style={{ backgroundImage: cinemaGradient }}>
      <header className="cp-customer__header">
        <Link to={PATHS.home} className="cp-customer__brand">
          <span className="cp-customer__brand-mark" aria-hidden="true">
            ◗
          </span>
          {t('customer.brand')}
        </Link>

        <nav className="cp-customer__nav">
          {isCustomer ? (
            <NavLink
              to={PATHS.myTickets}
              className={({ isActive }) =>
                `cp-customer__link${isActive ? ' cp-customer__link--active' : ''}`
              }
            >
              {t('customer.myTickets')}
            </NavLink>
          ) : null}

          {isAuthenticated ? (
            <button type="button" className="cp-btn cp-btn--danger" onClick={handleLogout}>
              {t('common.logout')}
            </button>
          ) : (
            <>
              <Link to={PATHS.login} className="cp-btn cp-btn--primary">
                {t('customer.login')}
              </Link>
              <Link to={PATHS.register} className="cp-btn cp-btn--ghost">
                {t('customer.register')}
              </Link>
            </>
          )}
        </nav>
      </header>

      <main className="cp-customer__main">
        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>
      </main>

      <footer className="cp-customer__footer">{t('customer.footer')}</footer>
    </div>
  );
};

export default CustomerLayout;
