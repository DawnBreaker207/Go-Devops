import { useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import FieldInput from '@/components/ui/FieldInput';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { authApi } from '@/api/auth.api';
import { PATHS } from '@/routes/paths';
import { errorMessage, fieldErrorsOf } from '@/utils/error';

/** Step 2: reset via ?token= from the email link. */
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
      navigate(PATHS.customerLogin, { state: { passwordReset: true } });
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
        <Link to={PATHS.customerLogin} className="font-semibold text-brand-active no-underline">
          {t('customer.backToLogin')}
        </Link>
      }
    >
      {!token ? (
        <Notice variant="error">{t('customer.resetNoToken')}</Notice>
      ) : (
        <form onSubmit={submit} noValidate>
          {error ? (
            <Notice variant="error" className="mb-3.5">
              {error}
            </Notice>
          ) : null}

          <FieldInput
            id="new_password"
            label={t('customer.newPassword')}
            type="password"
            required
            minLength={6}
            maxLength={72}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            error={fieldErrors.new_password}
          />

          <FieldInput
            id="confirm"
            label={t('customer.confirmPassword')}
            type="password"
            required
            value={confirm}
            onChange={(event) => setConfirm(event.target.value)}
            error={fieldErrors.confirm}
          />

          <Button type="submit" variant="primary" block className="mt-2" disabled={busy}>
            {busy ? t('common.loading') : t('customer.resetSubmit')}
          </Button>
        </form>
      )}
    </CustomerAuthShell>
  );
};

export default ResetPasswordPage;
