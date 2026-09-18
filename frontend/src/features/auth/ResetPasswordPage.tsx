import { useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import { authApi } from '@/api/auth.api';
import { PATHS } from '@/routes/paths';
import { errorMessage, fieldErrorsOf } from '@/utils/error';

/**
 * Quen mat khau - buoc 2: dat lai bang token trong link email.
 *
 * Token den tu query `?token=...`. Luu y trong `.claude/context/cross-repo-gotchas.md`:
 * `ACCOUNT_PASSWORD_RESET_URL` cua backend dang tro toi cong 5173 trong khi
 * Vite chay o 3000, nen link trong email hien KHONG mo duoc man nay - phai dan
 * token vao tay hoac sua bien do. Man nay van dung khi token duoc dua vao URL.
 */
export const ResetPasswordPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';

  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setFieldErrors({});

    if (password !== confirm) {
      setFieldErrors({ confirm: t('customer.passwordMismatch') });
      return;
    }

    setBusy(true);
    try {
      await authApi.resetPassword(token, password);
      navigate(PATHS.login, { state: { passwordReset: true } });
    } catch (err) {
      const details = fieldErrorsOf(err);
      if (details) setFieldErrors(details);
      else setError(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setBusy(false);
    }
  };

  return (
    <CustomerAuthShell
      title={t('customer.resetTitle')}
      foot={
        <Link to={PATHS.login} className="cp-auth__link">
          {t('customer.backToLogin')}
        </Link>
      }
    >
      {!token ? (
        <div className="cp-auth__notice cp-auth__notice--error">{t('customer.resetNoToken')}</div>
      ) : (
        <form onSubmit={submit} noValidate>
          {error ? (
            <div className="cp-auth__notice cp-auth__notice--error" role="alert">
              {error}
            </div>
          ) : null}

          <div className="cp-field">
            <label className="cp-field__label" htmlFor="new_password">
              {t('customer.newPassword')}
            </label>
            <input
              id="new_password"
              type="password"
              required
              minLength={6}
              maxLength={72}
              className="cp-field__input"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
            {fieldErrors.new_password ? (
              <span className="cp-field__error">{fieldErrors.new_password}</span>
            ) : null}
          </div>

          <div className="cp-field">
            <label className="cp-field__label" htmlFor="confirm">
              {t('customer.confirmPassword')}
            </label>
            <input
              id="confirm"
              type="password"
              required
              className="cp-field__input"
              value={confirm}
              onChange={(event) => setConfirm(event.target.value)}
            />
            {fieldErrors.confirm ? (
              <span className="cp-field__error">{fieldErrors.confirm}</span>
            ) : null}
          </div>

          <button type="submit" className="cp-btn cp-btn--primary cp-auth__submit" disabled={busy}>
            {busy ? t('common.loading') : t('customer.resetSubmit')}
          </button>
        </form>
      )}
    </CustomerAuthShell>
  );
};

export default ResetPasswordPage;
