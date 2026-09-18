import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import { authApi } from '@/api/auth.api';
import { PATHS } from '@/routes/paths';
import { errorMessage } from '@/utils/error';

/**
 * Quen mat khau - buoc 1: xin link dat lai.
 *
 * Backend tra 200 DU email co ton tai hay khong, de khong lo ra email nao da
 * dang ky. Vi vay man nay KHONG duoc noi "da gui" theo kieu khang dinh email
 * ton tai; cau chu phai trung tinh.
 */
export const ForgotPasswordPage = () => {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await authApi.forgotPassword(email.trim());
      setSent(true);
    } catch (err) {
      setError(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setBusy(false);
    }
  };

  return (
    <CustomerAuthShell
      title={t('customer.forgotTitle')}
      foot={
        <Link to={PATHS.login} className="cp-auth__link">
          {t('customer.backToLogin')}
        </Link>
      }
    >
      {sent ? (
        <div className="cp-auth__notice cp-auth__notice--ok">{t('customer.forgotSent')}</div>
      ) : (
        <form onSubmit={submit} noValidate>
          {error ? (
            <div className="cp-auth__notice cp-auth__notice--error" role="alert">
              {error}
            </div>
          ) : null}

          <p style={{ fontSize: 13, color: '#666', marginTop: 0 }}>{t('customer.forgotHint')}</p>

          <div className="cp-field">
            <label className="cp-field__label" htmlFor="email">
              {t('user.email')}
            </label>
            <input
              id="email"
              type="email"
              required
              className="cp-field__input"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>

          <button type="submit" className="cp-btn cp-btn--primary cp-auth__submit" disabled={busy}>
            {busy ? t('common.loading') : t('customer.sendResetLink')}
          </button>
        </form>
      )}
    </CustomerAuthShell>
  );
};

export default ForgotPasswordPage;
