import { useState } from 'react';
import { useTranslation } from 'react-i18next';

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
