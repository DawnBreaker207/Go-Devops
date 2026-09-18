import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { cinemaGradientPanel } from '@/theme';
import { PATHS } from '@/routes/paths';
import './CustomerAuthPage.css';

interface CustomerAuthShellProps {
  title: string;
  children: ReactNode;
  foot?: ReactNode;
}

/**
 * Vo chia doi cua cac man tai khoan phia KHACH. Figma frame Sign In (77-626).
 *
 * Khac han `AuthLayout` cua khu van hanh (mot the can giua): khu khach co nua
 * trai la gradient toi mang loi chao, nua phai la form tren nen trang.
 */
export const CustomerAuthShell = ({ title, children, foot }: CustomerAuthShellProps) => {
  const { t } = useTranslation();

  return (
    <div className="cp-auth">
      <aside className="cp-auth__side" style={{ backgroundImage: cinemaGradientPanel }}>
        <Link to={PATHS.home} className="cp-auth__brand">
          <span className="cp-auth__brand-mark" aria-hidden="true">
            ◗
          </span>
          {t('customer.brand')}
        </Link>
        <p className="cp-auth__welcome">{t('customer.welcome')}</p>
        <span />
      </aside>

      <main className="cp-auth__panel">
        <div className="cp-auth__form">
          <h1 className="cp-auth__title">{title}</h1>
          {children}
          {foot ? <p className="cp-auth__foot">{foot}</p> : null}
        </div>
      </main>
    </div>
  );
};

export default CustomerAuthShell;
