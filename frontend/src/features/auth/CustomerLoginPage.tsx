import { useLocation, useNavigate, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import CustomerLoginForm from './CustomerLoginForm';
import Notice from '@/components/ui/Notice';
import { PATHS } from '@/routes/paths';

interface LocationState {
  from?: string;
  registered?: boolean;
  passwordReset?: boolean;
}

export const CustomerLoginPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();

  const state = (location.state as LocationState | null) ?? null;

  return (
    <CustomerAuthShell
      title={t('customer.login')}
      foot={
        <>
          {t('customer.noAccount')}{' '}
          <Link to={PATHS.register} className="font-semibold text-brand-active no-underline">
            {t('customer.createAccount')}
          </Link>
        </>
      }
    >
      {state?.registered ? (
        <Notice variant="success" role="status" className="mb-3.5">
          {t('customer.registerSuccessNotice')}
        </Notice>
      ) : null}

      {state?.passwordReset ? (
        <Notice variant="success" role="status" className="mb-3.5">
          {t('customer.resetSuccessNotice')}
        </Notice>
      ) : null}

      <CustomerLoginForm
        idPrefix="cp-login-page"
        onLoggedIn={(landingPath) => navigate(state?.from ?? landingPath, { replace: true })}
        footerSlot={
          <div className="-mt-1.5 mb-3.5 text-right text-[13px]">
            <Link
              to={PATHS.forgotPassword}
              className="font-semibold text-brand-active no-underline"
            >
              {t('customer.forgotTitle')}
            </Link>
          </div>
        }
      />
    </CustomerAuthShell>
  );
};

export default CustomerLoginPage;
