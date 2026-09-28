import { useState, type FormEvent, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { performLogin } from './loginFlow';
import FieldInput from '@/components/ui/FieldInput';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { isValidEmail } from '@/utils/validators';

export interface CustomerLoginFormProps {
  onLoggedIn: (landingPath: string) => void;
  autoFocus?: boolean;
  idPrefix: string;
  footerSlot?: ReactNode;
}

// Shared by page + sheet shells, only the shell differs.
export const CustomerLoginForm = ({
  onLoggedIn,
  autoFocus,
  idPrefix,
  footerSlot,
}: CustomerLoginFormProps) => {
  const { t } = useTranslation();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<{ email?: string; password?: string }>({});
  const [busy, setBusy] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);

    const nextFieldErrors: { email?: string; password?: string } = {};
    if (!email.trim()) nextFieldErrors.email = t('auth.emailRequired');
    else if (!isValidEmail(email.trim())) nextFieldErrors.email = t('auth.emailInvalid');
    if (!password) nextFieldErrors.password = t('auth.passwordRequired');
    // Matches the backend min=6 binding, else a generic English error comes back.
    else if (password.length < 6) nextFieldErrors.password = t('auth.passwordMin');
    setFieldErrors(nextFieldErrors);
    if (Object.keys(nextFieldErrors).length > 0) return;

    setBusy(true);
    const result = await performLogin({ email: email.trim(), password }, t);
    setBusy(false);
    if (result.ok) {
      onLoggedIn(result.landingPath);
    } else if (result.fieldErrors) {
      setFieldErrors((current) => ({ ...current, ...result.fieldErrors }));
    } else {
      setError(result.message);
    }
  };

  return (
    <form onSubmit={submit} noValidate>
      {error ? (
        <Notice variant="error" className="mb-3.5">
          {error}
        </Notice>
      ) : null}

      <FieldInput
        id={`${idPrefix}-email`}
        label={t('user.email')}
        type="email"
        required
        placeholder="ban@email.com"
        autoFocus={autoFocus}
        value={email}
        onChange={(event) => {
          setEmail(event.target.value);
          if (fieldErrors.email) setFieldErrors((current) => ({ ...current, email: undefined }));
        }}
        error={fieldErrors.email}
      />

      <FieldInput
        id={`${idPrefix}-password`}
        label={t('user.password')}
        type="password"
        required
        placeholder="••••••"
        value={password}
        onChange={(event) => {
          setPassword(event.target.value);
          if (fieldErrors.password)
            setFieldErrors((current) => ({ ...current, password: undefined }));
        }}
        error={fieldErrors.password}
      />

      {footerSlot}

      <Button type="submit" variant="primary" block className="mt-2" disabled={busy}>
        {busy ? t('common.loading') : t('customer.login')}
      </Button>
    </form>
  );
};

export default CustomerLoginForm;
