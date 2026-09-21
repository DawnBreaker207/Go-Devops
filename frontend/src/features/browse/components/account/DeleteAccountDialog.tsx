import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDeleteAccount } from '@/hooks/useAccount';
import { errorMessage, isApiError } from '@/utils/error';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import FieldInput from '@/components/ui/FieldInput';

export interface DeleteAccountDialogProps {
  onClose: () => void;
  /** Called ONLY when `DELETE /users/me` succeeds (204). The parent page owns
   *  logout() + navigating home - this dialog knows nothing about the router. */
  onDeleted: () => void;
}

/** Short countdown before enabling delete confirmation. */
const CONFIRM_COUNTDOWN_SECONDS = 10;

/** Delete-account dialog requiring password confirmation. */
export const DeleteAccountDialog = ({ onClose, onDeleted }: DeleteAccountDialogProps) => {
  const { t } = useTranslation();
  const deleteAccount = useDeleteAccount();

  const [password, setPassword] = useState('');
  const [secondsLeft, setSecondsLeft] = useState(CONFIRM_COUNTDOWN_SECONDS);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (secondsLeft <= 0) return;
    const timer = window.setTimeout(() => setSecondsLeft((s) => s - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [secondsLeft]);

  const canConfirm = secondsLeft <= 0 && password.length > 0 && !deleteAccount.isPending;

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!canConfirm) return;
    setFormError(null);
    try {
      await deleteAccount.mutateAsync({ password });
      onDeleted();
    } catch (error) {
      if (isApiError(error) && error.code === 40900) {
        setFormError(t('customer.accountDeleteHasTickets'));
      } else if (isApiError(error) && (error.code === 401 || error.code === 40100)) {
        setFormError(t('customer.accountDeleteWrongPassword'));
      } else {
        setFormError(errorMessage(error, t('common.somethingWrong')));
      }
    }
  };

  return (
    <div
      className="fixed inset-0 z-1000 flex items-center justify-center bg-black/60 p-4"
      role="presentation"
      onClick={onClose}
    >
      <div
        className="w-full max-w-105 rounded-[14px] bg-(--cp-surface-base) p-6 text-[#141414]"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="cp-acc-dialog-title"
        onClick={(event) => event.stopPropagation()}
      >
        <h2 id="cp-acc-dialog-title" className="mt-0 mb-2.5 text-lg font-bold text-danger">
          {t('customer.accountDeleteTitle')}
        </h2>
        <p className="mt-0 mb-2 text-[13px] leading-[1.6] text-[#333]">
          {t('customer.accountDeleteWarning')}
        </p>
        <ul className="mt-0 mb-3.5 pl-4.5 text-[13px] leading-[1.6] text-[#333]">
          <li>{t('customer.accountDeleteConsequence1')}</li>
          <li>{t('customer.accountDeleteConsequence2')}</li>
          <li>{t('customer.accountDeleteConsequence3')}</li>
        </ul>

        {formError ? (
          <Notice variant="error" className="mb-4">
            {formError}
          </Notice>
        ) : null}

        <form onSubmit={submit} noValidate>
          <FieldInput
            id="acc-delete-password"
            label={t('user.password')}
            type="password"
            required
            autoFocus
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />

          <div className="mt-4.5 flex gap-2.5">
            <Button
              type="button"
              variant="ghost"
              className="flex-1"
              onClick={onClose}
              disabled={deleteAccount.isPending}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" variant="danger" className="flex-1" disabled={!canConfirm}>
              {deleteAccount.isPending
                ? t('common.loading')
                : secondsLeft > 0
                  ? t('customer.accountDeleteConfirmCountdown', { seconds: secondsLeft })
                  : t('customer.accountDeleteConfirm')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default DeleteAccountDialog;
