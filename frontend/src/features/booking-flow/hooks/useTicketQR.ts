import { useEffect, useState } from 'react';
import { ticketApi } from '@/api/ticket.api';
import { getCachedQR, setCachedQR } from '@/utils/qrCache';

/** One ticket's QR: offline cache first (instant), silent refresh after. Offline still shows cache; `ticketId` is stable per card so the effect runs once. */
export const useTicketQR = (ticketId: string | undefined) => {
  const [qrBase64, setQrBase64] = useState<string | null>(null);
  // "Loading" only while NOTHING is showable yet (neither cache nor network done).
  const [loading, setLoading] = useState(Boolean(ticketId));

  useEffect(() => {
    if (!ticketId) return;

    let cancelled = false;

    getCachedQR(ticketId).then((cached) => {
      if (cancelled || !cached) return;
      setQrBase64(cached);
      setLoading(false);
    });

    ticketApi
      .qr(ticketId)
      .then((res) => {
        if (cancelled) return;
        setQrBase64(res.qr_base64);
        setLoading(false);
        void setCachedQR(ticketId, res.qr_base64);
      })
      .catch(() => {
        // Offline/server error: shown cache (if any) stands, never overwritten
        // by the error. Only clear "loading" when still nothing at all.
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [ticketId]);

  return { qrBase64, loading };
};
