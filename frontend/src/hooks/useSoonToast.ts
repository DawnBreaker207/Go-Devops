import { useState } from 'react';
import { useTranslation } from 'react-i18next';

/** Shared stub toast for not-yet-built actions; auto-dismisses after 2s, calls no API. */
export const useSoonToast = () => {
  const { t } = useTranslation();
  const [message, setMessage] = useState<string | null>(null);

  const showSoon = () => {
    setMessage(t('customer.soon'));
    window.setTimeout(() => setMessage(null), 2000);
  };

  return { toastMessage: message, showSoon };
};

export default useSoonToast;
