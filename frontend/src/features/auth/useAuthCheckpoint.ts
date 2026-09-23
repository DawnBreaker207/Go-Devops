import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { ACCESS_TOKEN_STORAGE_KEY, tokenStorage } from '@/utils/storage';

/** Login gate running a pending action after sign-in, synced across tabs. */
export const useAuthCheckpoint = () => {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const bootstrap = useAuthStore((s) => s.bootstrap);

  const [sheetOpen, setSheetOpen] = useState(false);
  const [contextMessage, setContextMessage] = useState('');
  // Ref, not state: the pending action needs no re-render when it changes.
  const pendingAction = useRef<(() => void) | null>(null);

  const runPendingAction = useCallback(() => {
    const action = pendingAction.current;
    pendingAction.current = null;
    setSheetOpen(false);
    action?.();
  }, []);

  const requireAuth = useCallback(
    (action: () => void, message: string) => {
      if (isAuthenticated) {
        action();
        return;
      }
      pendingAction.current = action;
      setContextMessage(message);
      setSheetOpen(true);
    },
    [isAuthenticated]
  );

  const closeSheet = useCallback(() => {
    pendingAction.current = null;
    setSheetOpen(false);
  }, []);

  // Signed in successfully INSIDE this sheet -> useAuthStore already set token +
  // isAuthenticated, just run the pending action and close the sheet.
  const handleSheetSuccess = useCallback(() => {
    runPendingAction();
  }, [runPendingAction]);

  // Signed in from ANOTHER TAB while this sheet waits: listen for
  // `storage` on the exact key tokenStorage uses (cp_access_token), re-bootstrap
  // this tab's authStore, then run the pending action.
  useEffect(() => {
    if (!sheetOpen) return;

    const onStorage = (event: StorageEvent) => {
      if (event.key !== ACCESS_TOKEN_STORAGE_KEY || !event.newValue) return;
      void bootstrap().then(() => {
        if (tokenStorage.getAccessToken()) {
          runPendingAction();
        }
      });
    };

    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [sheetOpen, bootstrap, runPendingAction]);

  return { requireAuth, sheetOpen, contextMessage, closeSheet, handleSheetSuccess };
};
