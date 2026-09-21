import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useUpdateProfile } from '@/hooks/useAccount';
import { errorMessage, fieldErrorsOf } from '@/utils/error';
import type { User } from '@/types';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import FieldInput from '@/components/ui/FieldInput';
import { INK_65 } from '@/theme/customerTw';

interface ProfileSectionProps {
  user: User;
}

/** Profile form editing name and phone; email stays read-only. */
export const ProfileSection = ({ user }: ProfileSectionProps) => {
  const { t } = useTranslation();
  const updateProfile = useUpdateProfile();

  const [fullName, setFullName] = useState(user.full_name);
  const [phone, setPhone] = useState(user.phone ?? '');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [savedNotice, setSavedNotice] = useState(false);

  // Props-seeded state goes stale when `user` changes underneath (edited in
  // another tab). Resync during render while locally untouched; dirty input
  // always wins over the incoming snapshot.
  const [synced, setSynced] = useState({ name: user.full_name, phone: user.phone ?? '' });
  if (user.full_name !== synced.name || (user.phone ?? '') !== synced.phone) {
    if (fullName === synced.name && phone === synced.phone) {
      setFullName(user.full_name);
      setPhone(user.phone ?? '');
    }
    setSynced({ name: user.full_name, phone: user.phone ?? '' });
  }

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});
    setSavedNotice(false);
    try {
      await updateProfile.mutateAsync({ full_name: fullName.trim(), phone: phone.trim() });
      setSavedNotice(true);
    } catch (error) {
      const details = fieldErrorsOf(error);
      if (details) setFieldErrors(details);
      else setFormError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  return (
    <section>
      <h2 className="mt-0 mb-1 text-base font-bold">{t('common.profile')}</h2>
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountProfileHint')}</p>

      {formError ? (
        <Notice variant="error" className="mb-4">
          {formError}
        </Notice>
      ) : null}

      {savedNotice ? (
        <Notice variant="info" className="mb-4">
          {t('common.updateSuccess')}
        </Notice>
      ) : null}

      <form onSubmit={submit} noValidate>
        <FieldInput
          id="acc-email"
          label={t('user.email')}
          type="email"
          value={user.email}
          disabled
          readOnly
        />

        <FieldInput
          id="acc-full-name"
          label={t('user.fullName')}
          required
          minLength={2}
          value={fullName}
          onChange={(event) => setFullName(event.target.value)}
          error={fieldErrors.full_name}
        />

        <FieldInput
          id="acc-phone"
          label={t('customer.accountPhone')}
          type="tel"
          maxLength={20}
          value={phone}
          onChange={(event) => setPhone(event.target.value)}
          error={fieldErrors.phone}
        />

        <div className="mt-1 flex gap-2.5">
          <Button type="submit" variant="primary" disabled={updateProfile.isPending}>
            {updateProfile.isPending ? t('common.loading') : t('common.save')}
          </Button>
        </div>
      </form>
    </section>
  );
};

export default ProfileSection;
