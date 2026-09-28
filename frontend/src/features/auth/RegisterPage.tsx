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
import { isValidEmail } from '@/utils/validators';

// Register forces the customer role and answers 201 with no token, so go to login after.
export const RegisterPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [values, setValues] = useState({ email: '', full_name: '', password: '', confirm: '' });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string | undefined>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const set = (key: keyof typeof values) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setValues((current) => ({ ...current, [key]: event.target.value }));
    if (fieldErrors[key]) setFieldErrors((current) => ({ ...current, [key]: undefined }));
  };

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setFormError(null);

    const nextFieldErrors: Record<string, string> = {};
    if (!values.email.trim()) nextFieldErrors.email = t('auth.emailRequired');
    else if (!isValidEmail(values.email.trim())) nextFieldErrors.email = t('auth.emailInvalid');
    if (!values.full_name.trim()) nextFieldErrors.full_name = t('common.requiredField');
    if (!values.password) nextFieldErrors.password = t('auth.passwordRequired');
    if (!values.confirm) nextFieldErrors.confirm = t('common.requiredField');
    if (!nextFieldErrors.confirm && values.password !== values.confirm) {
      // Confirm-password exists only in the form; the backend has none, so check here.
      nextFieldErrors.confirm = t('customer.passwordMismatch');
    }
    setFieldErrors(nextFieldErrors);
    if (Object.keys(nextFieldErrors).length > 0) return;

    setBusy(true);
    try {
      await authApi.register({
        email: values.email.trim(),
        password: values.password,
        full_name: values.full_name.trim(),
      });
      navigate(PATHS.customerLogin, { state: { registered: true } });
    } catch (error) {
      // Details follow the DTO json names; duplicate email is 409 with no details.
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
          placeholder="ban@email.com"
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
          placeholder="Nguyen Van A"
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
          placeholder="••••••"
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
          placeholder="••••••"
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
