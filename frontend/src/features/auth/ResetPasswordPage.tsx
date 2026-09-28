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

export const ResetPasswordPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';

  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string | undefined>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);

    const nextFieldErrors: Record<string, string> = {};
    if (!password) nextFieldErrors.new_password = t('auth.passwordRequired');
    if (!confirm) nextFieldErrors.confirm = t('common.requiredField');
    if (!nextFieldErrors.confirm && password !== confirm) {
      nextFieldErrors.confirm = t('customer.passwordMismatch');
    }
    setFieldErrors(nextFieldErrors);
    if (Object.keys(nextFieldErrors).length > 0) return;

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
            placeholder="••••••"
            value={password}
            onChange={(event) => {
              setPassword(event.target.value);
              if (fieldErrors.new_password)
                setFieldErrors((current) => ({ ...current, new_password: undefined }));
            }}
            error={fieldErrors.new_password}
          />

          <FieldInput
            id="confirm"
            label={t('customer.confirmPassword')}
            type="password"
            required
            placeholder="••••••"
            value={confirm}
            onChange={(event) => {
              setConfirm(event.target.value);
              if (fieldErrors.confirm)
                setFieldErrors((current) => ({ ...current, confirm: undefined }));
            }}
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
