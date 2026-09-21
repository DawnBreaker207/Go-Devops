import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import FieldInput from '@/components/ui/FieldInput';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import { authApi } from '@/api/auth.api';
import { PATHS } from '@/routes/paths';
import { errorMessage } from '@/utils/error';

/** Step 1: request reset link. Backend is always 200 (no email enumeration), so copy must stay neutral about whether the email exists. */
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
        <Link to={PATHS.customerLogin} className="font-semibold text-brand-active no-underline">
          {t('customer.backToLogin')}
        </Link>
      }
    >
      {sent ? (
        <Notice variant="success" role="status">
          {t('customer.forgotSent')}
        </Notice>
      ) : (
        <form onSubmit={submit} noValidate>
          {error ? (
            <Notice variant="error" className="mb-3.5">
              {error}
            </Notice>
          ) : null}

          <p className="mt-0 mb-3.5 text-[13px] text-[#666]">{t('customer.forgotHint')}</p>

          <FieldInput
            id="email"
            label={t('user.email')}
            type="email"
            required
            value={email}
            onChange={(event) => setEmail(event.target.value)}
          />

          <Button type="submit" variant="primary" block className="mt-2" disabled={busy}>
            {busy ? t('common.loading') : t('customer.sendResetLink')}
          </Button>
        </form>
      )}
    </CustomerAuthShell>
  );
};

export default ForgotPasswordPage;
