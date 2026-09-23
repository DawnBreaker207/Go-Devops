import { useState, type FormEvent, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { performLogin } from './loginFlow';
import FieldInput from '@/components/ui/FieldInput';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';

export interface CustomerLoginFormProps {
  /** Called on success with the role-based landing path; caller decides navigate vs close-sheet/continue. */
  onLoggedIn: (landingPath: string) => void;
  autoFocus?: boolean;
  /** id prefix for label/input pairs; avoids duplicate DOM ids when page and sheet coexist. */
  idPrefix: string;
  /** Slot between inputs and submit, e.g. the "Forgot password" link (page only, not the sheet). */
  footerSlot?: ReactNode;
}

/** Plain-CSS email/password form shared by CustomerLoginPage and LoginBottomSheet. Only the shell (full page vs overlay) and post-login action differ. */
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
  const [busy, setBusy] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    setBusy(true);
    const result = await performLogin({ email: email.trim(), password }, t('auth.loginFailed'));
    setBusy(false);
    if (result.ok) {
      onLoggedIn(result.landingPath);
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
        autoFocus={autoFocus}
        value={email}
        onChange={(event) => setEmail(event.target.value)}
      />

      <FieldInput
        id={`${idPrefix}-password`}
        label={t('user.password')}
        type="password"
        required
        value={password}
        onChange={(event) => setPassword(event.target.value)}
      />

      {footerSlot}

      <Button type="submit" variant="primary" block className="mt-2" disabled={busy}>
        {busy ? t('common.loading') : t('customer.login')}
      </Button>
    </form>
  );
};

export default CustomerLoginForm;
