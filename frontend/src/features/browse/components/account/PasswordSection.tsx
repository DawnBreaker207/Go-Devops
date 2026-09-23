import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import Button from '@/components/ui/Button';
import FieldInput from '@/components/ui/FieldInput';
import Notice from '@/components/ui/Notice';
import { useAuthStore } from '@/stores/authStore';
import { errorMessage } from '@/utils/error';
import { INK_65 } from '@/theme/customerTw';

/** Backend binding on ChangePasswordRequest.new_password. */
const PASSWORD_MIN = 6;
const PASSWORD_MAX = 72;

/** Change password from the account screen.
 *
 *  Until this existed the only way to change a password was the forgot-password
 *  link, which goes out through a mock mailer that writes a file on the server
 *  and never reaches anyone — so an account's password was effectively fixed. */
export const PasswordSection = () => {
  const { t } = useTranslation();
  const changePassword = useAuthStore((s) => s.changePassword);

  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const tooShort = next.length > 0 && next.length < PASSWORD_MIN;
  const mismatch = confirm.length > 0 && confirm !== next;
  const canSubmit =
    current.length > 0 && next.length >= PASSWORD_MIN && confirm === next && !submitting;

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!canSubmit) return;
    setError(null);
    setDone(false);
    setSubmitting(true);
    try {
      await changePassword(current, next);
      setCurrent('');
      setNext('');
      setConfirm('');
      setDone(true);
    } catch (err) {
      // A wrong current password is 401 with no `details` map, so there is no
      // field to bind it to - it can only be shown as a message.
      setError(errorMessage(err, t('common.somethingWrong')));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={(event) => void submit(event)} className="max-w-100">
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.passwordSignsOutOthers')}</p>

      {error ? <Notice variant="error">{error}</Notice> : null}
      {done ? <Notice variant="success">{t('customer.passwordChanged')}</Notice> : null}

      <FieldInput
        id="account-current-password"
        type="password"
        autoComplete="current-password"
        label={t('customer.passwordCurrent')}
        value={current}
        onChange={(event) => {
          setCurrent(event.target.value);
          setError(null);
        }}
      />
      <FieldInput
        id="account-new-password"
        type="password"
        autoComplete="new-password"
        maxLength={PASSWORD_MAX}
        label={t('customer.passwordNew')}
        hint={t('user.passwordHint')}
        error={tooShort ? t('customer.passwordTooShort', { min: PASSWORD_MIN }) : undefined}
        value={next}
        onChange={(event) => {
          setNext(event.target.value);
          setError(null);
        }}
      />
      <FieldInput
        id="account-confirm-password"
        type="password"
        autoComplete="new-password"
        maxLength={PASSWORD_MAX}
        label={t('customer.passwordConfirm')}
        error={mismatch ? t('customer.passwordMismatch') : undefined}
        value={confirm}
        onChange={(event) => {
          setConfirm(event.target.value);
          setError(null);
        }}
      />

      <Button type="submit" disabled={!canSubmit}>
        {submitting ? t('common.loading') : t('common.save')}
      </Button>
    </form>
  );
};

export default PasswordSection;
