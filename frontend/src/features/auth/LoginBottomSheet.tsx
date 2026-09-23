import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import CustomerLoginForm from './CustomerLoginForm';
import { PATHS } from '@/routes/paths';

export interface LoginBottomSheetProps {
  /** Contextual prompt, e.g. "Sign in to pick seats for the 20:00 show at X". */
  contextMessage: string;
  onClose: () => void;
  /** Fired only on in-sheet login success (including login from another tab; see useAuthCheckpoint). */
  onSuccess: () => void;
}

/** Login checkpoint as a bottom sheet over a dim backdrop (keeps the selected show). Shares CustomerLoginForm with the full page, only the shell differs. Mounted only while open so the form is fresh each time, no reset effect. Slide/fade on mount uses `transition` (not keyframes) so prefers-reduced-motion via --motion-duration-* applies; only transform/opacity animate. */
export const LoginBottomSheet = ({ contextMessage, onClose, onSuccess }: LoginBottomSheetProps) => {
  const { t } = useTranslation();

  const [shown, setShown] = useState(false);
  useEffect(() => {
    const raf = requestAnimationFrame(() => setShown(true));
    return () => cancelAnimationFrame(raf);
  }, []);

  return (
    <div
      role="presentation"
      onClick={onClose}
      className={`fixed inset-0 z-1000 flex items-end justify-center bg-black/60 transition-opacity duration-moderate ease-out ${
        shown ? 'opacity-100' : 'opacity-0'
      }`}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cp-sheet-title"
        onClick={(event) => event.stopPropagation()}
        className={`relative max-h-[90vh] w-full max-w-105 overflow-y-auto rounded-t-2xl bg-(--cp-surface-base) px-6 pt-3 pb-7 text-[#141414] shadow-[0_-8px_32px_rgba(0,0,0,0.35)] transition-transform duration-moderate ease-out sm:mb-10 sm:rounded-2xl ${
          shown ? 'translate-y-0' : 'translate-y-6'
        }`}
      >
        <div
          className="mx-auto mt-2 mb-1 h-1 w-10 rounded-full bg-(--cp-surface-border)"
          aria-hidden="true"
        />

        <button
          type="button"
          aria-label={t('common.cancel')}
          onClick={onClose}
          className="absolute top-3 right-4 cursor-pointer border-none bg-transparent p-1.5 text-base leading-none text-[#666]"
        >
          ✕
        </button>

        <h2 id="cp-sheet-title" className="mt-2 mb-1 text-[22px] font-bold">
          {t('customer.login')}
        </h2>
        <p className="mt-0 mb-4.5 text-[13px] text-[#555]">{contextMessage}</p>

        <CustomerLoginForm idPrefix="cp-sheet" autoFocus onLoggedIn={() => onSuccess()} />

        <p className="mt-4 text-center text-[13px] text-[#666]">
          {t('customer.noAccount')}{' '}
          <Link
            to={PATHS.register}
            className="font-semibold text-brand-active no-underline"
            onClick={onClose}
          >
            {t('customer.createAccount')}
          </Link>
        </p>
      </div>
    </div>
  );
};

export default LoginBottomSheet;
