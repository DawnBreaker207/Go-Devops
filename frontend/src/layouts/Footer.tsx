import { useState } from 'react';
import type { FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PATHS } from '@/routes/paths';
import { useSoonToast } from '@/hooks/useSoonToast';
import SoonToast from '@/components/ui/SoonToast';
import { FOCUS_RING, TRANSITION_FAST } from '@/theme/customerTw';

/** Footer: 4 columns + copyright bar, always dark. Pageless links share the "coming soon" toast. */
// App-wide preflight is off (see index.css), so reset button/list here.
// Links always stand out (brand red), never change on hover.
const linkClass = `border-none bg-transparent p-0 text-left text-[13px] font-semibold text-brand no-underline ${TRANSITION_FAST}`;

export const Footer = () => {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [subscribed, setSubscribed] = useState(false);
  const { toastMessage, showSoon: soon } = useSoonToast();

  const onSubscribe = (event: FormEvent) => {
    event.preventDefault();
    if (!email.trim()) return;
    setSubscribed(true);
  };

  return (
    <footer className="relative border-t border-white/10 bg-[#141414] pt-14 pb-7 text-white/70">
      <div className="mx-auto grid max-w-300 grid-cols-1 gap-10 px-4 sm:px-8 md:grid-cols-4 md:gap-12">
        <div className="flex flex-col items-start gap-4">
          <div className="inline-flex items-center gap-2 text-xl font-bold text-white">
            <span className="text-2xl leading-none text-brand" aria-hidden="true">
              ◗
            </span>
            {t('customer.brand')}
          </div>
          <p className="text-[13px] text-white/50">{t('customer.footerAboutText')}</p>
          <div className="flex gap-3 pt-1">
            <button
              type="button"
              aria-label={t('customer.footerFacebook')}
              onClick={soon}
              className={`flex h-10 w-10 items-center justify-center rounded-full border-none bg-white/10 p-0 text-white ${TRANSITION_FAST} hover-fine:bg-brand ${FOCUS_RING}`}
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="currentColor"
                aria-hidden="true"
              >
                <path d="M22 12.06C22 6.5 17.52 2 12 2S2 6.5 2 12.06c0 5.02 3.66 9.18 8.44 9.94v-7.03H7.9v-2.91h2.54V9.85c0-2.5 1.49-3.89 3.77-3.89 1.09 0 2.23.2 2.23.2v2.46h-1.26c-1.24 0-1.63.77-1.63 1.56v1.88h2.78l-.44 2.91h-2.34V22c4.78-.76 8.44-4.92 8.44-9.94Z" />
              </svg>
            </button>
            <button
              type="button"
              aria-label={t('customer.footerGoogle')}
              onClick={soon}
              className={`flex h-10 w-10 items-center justify-center rounded-full border-none bg-white/10 p-0 text-[13px] font-bold text-white ${TRANSITION_FAST} hover-fine:bg-brand ${FOCUS_RING}`}
            >
              G
            </button>
          </div>
        </div>

        <div>
          <h4 className="mb-5 text-sm font-bold text-white">{t('customer.footerQuickLinks')}</h4>
          <ul className="m-0 flex list-none flex-col gap-3 p-0 text-[13px]">
            <li>
              <button type="button" onClick={soon} className={linkClass}>
                {t('customer.footerAbout')}
              </button>
            </li>
            <li>
              <Link to={PATHS.films} className={linkClass}>
                {t('customer.navMovies')}
              </Link>
            </li>
            <li>
              <Link to={PATHS.cinemaInfo} className={linkClass}>
                {t('customer.navCinema')}
              </Link>
            </li>
            <li>
              <button type="button" onClick={soon} className={linkClass}>
                {t('customer.footerContact')}
              </button>
            </li>
          </ul>
        </div>

        <div>
          <h4 className="mb-5 text-sm font-bold text-white">{t('customer.footerSupport')}</h4>
          <ul className="m-0 flex list-none flex-col gap-3 p-0 text-[13px]">
            <li>
              <button type="button" onClick={soon} className={linkClass}>
                {t('customer.footerPrivacy')}
              </button>
            </li>
            <li>
              <button type="button" onClick={soon} className={linkClass}>
                {t('customer.footerTerms')}
              </button>
            </li>
            <li>
              <button type="button" onClick={soon} className={linkClass}>
                {t('customer.footerFaq')}
              </button>
            </li>
            <li>
              <button type="button" onClick={soon} className={linkClass}>
                {t('customer.footerSupportCenter')}
              </button>
            </li>
          </ul>
        </div>

        <div>
          <h4 className="mb-5 text-sm font-bold text-white">{t('customer.footerNewsletter')}</h4>
          <p className="mb-4 text-xs leading-relaxed text-white/50">
            {t('customer.footerSubscribeText')}
          </p>
          {subscribed ? (
            <p className="text-sm text-[#4ade80]">{t('customer.footerSubscribed')}</p>
          ) : (
            <form className="flex gap-2" onSubmit={onSubscribe}>
              <label htmlFor="footer-newsletter-email" className="sr-only">
                {t('common.email')}
              </label>
              <input
                id="footer-newsletter-email"
                type="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder={t('customer.footerEmailPlaceholder')}
                className={`w-full rounded-l-lg border-none bg-white/10 px-4 py-2 text-sm text-white outline-none placeholder:text-white/35 ${FOCUS_RING}`}
              />
              <button
                type="submit"
                aria-label={t('customer.footerSubscribe')}
                className={`flex min-h-11 min-w-11 flex-none items-center justify-center rounded-r-lg border-none bg-brand px-4 py-2 font-bold text-on-brand ${TRANSITION_FAST} hover-fine:bg-brand-hover`}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                  <path
                    d="M3 6.5 12 13l9-6.5M4 5h16a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1Z"
                    stroke="currentColor"
                    strokeWidth="1.6"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </button>
            </form>
          )}
        </div>
      </div>

      <div className="mx-auto mt-10 max-w-300 border-t border-white/10 px-4 pt-6 text-center text-xs text-white/40 sm:px-8">
        {t('customer.footerRights', { year: new Date().getFullYear() })}
      </div>

      <SoonToast message={toastMessage} />
    </footer>
  );
};

export default Footer;
