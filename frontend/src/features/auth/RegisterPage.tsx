import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import FieldInput from '@/components/ui/FieldInput';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { authApi } from '@/api/auth.api';
import { PATHS } from '@/routes/paths';
import { errorMessage, fieldErrorsOf } from '@/utils/error';

/** Customer signup. POST /auth/register forces role `customer` and returns 201 with UserResponse but NO token, so redirect to login. Password: min 6, max 72 (bcrypt backend binding). */
export const RegisterPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [values, setValues] = useState({ email: '', full_name: '', password: '', confirm: '' });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const set = (key: keyof typeof values) => (event: React.ChangeEvent<HTMLInputElement>) =>
    setValues((current) => ({ ...current, [key]: event.target.value }));

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});

    if (values.password !== values.confirm) {
      // No "confirm password" concept server-side; form-only constraint, checked here not via 400.
      setFieldErrors({ confirm: t('customer.passwordMismatch') });
      return;
    }

    setBusy(true);
    try {
      await authApi.register({
        email: values.email.trim(),
        password: values.password,
        full_name: values.full_name.trim(),
      });
      navigate(PATHS.customerLogin, { state: { registered: true } });
    } catch (error) {
      // 400/40001 `details` keys are DTO json names matching these fields; duplicate email is 409 without details.
      const details = fieldErrorsOf(error);
      if (details) setFieldErrors(details);
      else setFormError(errorMessage(error, t('common.somethingWrong')));
    } finally {
      setBusy(false);
    }
  };

  return (
    <CustomerAuthShell
      title={t('customer.createAccount')}
      foot={
        <>
          {t('customer.haveAccount')}{' '}
          <Link to={PATHS.customerLogin} className="font-semibold text-brand-active no-underline">
            {t('customer.login')}
          </Link>
        </>
      }
    >
      <form onSubmit={submit} noValidate>
        {formError ? (
          <Notice variant="error" className="mb-3.5">
            {formError}
          </Notice>
        ) : null}

        <FieldInput
          id="email"
          name="email"
          label={t('user.email')}
          type="email"
          required
          value={values.email}
          onChange={set('email')}
          error={fieldErrors.email}
        />

        <FieldInput
          id="full_name"
          name="full_name"
          label={t('user.fullName')}
          required
          minLength={2}
          value={values.full_name}
          onChange={set('full_name')}
          error={fieldErrors.full_name}
        />

        <FieldInput
          id="password"
          name="password"
          label={t('user.password')}
          type="password"
          required
          minLength={6}
          maxLength={72}
          value={values.password}
          onChange={set('password')}
          error={fieldErrors.password}
        />

        <FieldInput
          id="confirm"
          name="confirm"
          label={t('customer.confirmPassword')}
          type="password"
          required
          value={values.confirm}
          onChange={set('confirm')}
          error={fieldErrors.confirm}
        />

        <Button type="submit" variant="primary" block className="mt-2" disabled={busy}>
          {busy ? t('common.loading') : t('customer.createAccount')}
        </Button>
      </form>
    </CustomerAuthShell>
  );
};

export default RegisterPage;
