import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CustomerAuthShell from './CustomerAuthShell';
import { authApi } from '@/api/auth.api';
import { PATHS } from '@/routes/paths';
import { errorMessage, fieldErrorsOf } from '@/utils/error';

/**
 * Dang ky tai khoan khach.
 *
 * `POST /auth/register` ep role thanh `customer` va tra ve 201 kem
 * UserResponse - KHONG kem token. Nen sau khi dang ky thanh cong phai dua sang
 * trang dang nhap chu khong the tu vao thang.
 *
 * Mat khau: min 6, max 72 (tran cua bcrypt, backend ep bang binding tag).
 */
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
      // Backend khong co khai niem "confirm password" - day la rang buoc cua
      // rieng form nay, nen kiem o day chu khong cho 400 tra ve.
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
      navigate(PATHS.login, { state: { registered: true } });
    } catch (error) {
      // 400/40001 tra ve `details` khoa theo TEN JSON TAG, khop dung ten field
      // cua form nay; con trung email la 409 khong co details.
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
          <Link to={PATHS.login} className="cp-auth__link">
            {t('customer.login')}
          </Link>
        </>
      }
    >
      <form onSubmit={submit} noValidate>
        {formError ? (
          <div className="cp-auth__notice cp-auth__notice--error" role="alert">
            {formError}
          </div>
        ) : null}

        <div className="cp-field">
          <label className="cp-field__label" htmlFor="email">
            {t('user.email')}
          </label>
          <input
            id="email"
            name="email"
            type="email"
            required
            className="cp-field__input"
            value={values.email}
            onChange={set('email')}
          />
          {fieldErrors.email ? <span className="cp-field__error">{fieldErrors.email}</span> : null}
        </div>

        <div className="cp-field">
          <label className="cp-field__label" htmlFor="full_name">
            {t('user.fullName')}
          </label>
          <input
            id="full_name"
            name="full_name"
            required
            minLength={2}
            className="cp-field__input"
            value={values.full_name}
            onChange={set('full_name')}
          />
          {fieldErrors.full_name ? (
            <span className="cp-field__error">{fieldErrors.full_name}</span>
          ) : null}
        </div>

        <div className="cp-field">
          <label className="cp-field__label" htmlFor="password">
            {t('user.password')}
          </label>
          <input
            id="password"
            name="password"
            type="password"
            required
            minLength={6}
            maxLength={72}
            className="cp-field__input"
            value={values.password}
            onChange={set('password')}
          />
          {fieldErrors.password ? (
            <span className="cp-field__error">{fieldErrors.password}</span>
          ) : null}
        </div>

        <div className="cp-field">
          <label className="cp-field__label" htmlFor="confirm">
            {t('customer.confirmPassword')}
          </label>
          <input
            id="confirm"
            name="confirm"
            type="password"
            required
            className="cp-field__input"
            value={values.confirm}
            onChange={set('confirm')}
          />
          {fieldErrors.confirm ? (
            <span className="cp-field__error">{fieldErrors.confirm}</span>
          ) : null}
        </div>

        <button type="submit" className="cp-btn cp-btn--primary cp-auth__submit" disabled={busy}>
          {busy ? t('common.loading') : t('customer.createAccount')}
        </button>
      </form>
    </CustomerAuthShell>
  );
};

export default RegisterPage;
