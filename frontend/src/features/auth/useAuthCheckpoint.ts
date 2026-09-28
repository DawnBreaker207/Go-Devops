import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { ACCESS_TOKEN_STORAGE_KEY, tokenStorage } from '@/utils/storage';

// Login gate holding a pending action, synced across tabs via storage events.
export const useAuthCheckpoint = () => {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const bootstrap = useAuthStore((s) => s.bootstrap);

  const [sheetOpen, setSheetOpen] = useState(false);
  const [contextMessage, setContextMessage] = useState('');
  // A ref holds the pending action; no re-render needed on change.
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

  // Sign in inside the sheet already updated the store; just run the pending action.
  const handleSheetSuccess = useCallback(() => {
    runPendingAction();
  }, [runPendingAction]);

  // A sign-in from another tab arrives via storage; reboot the store before running the action.
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
