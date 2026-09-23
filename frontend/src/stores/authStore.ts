import { create } from 'zustand';
import { authApi } from '@/api/auth.api';
import type { LoginRequest, User } from '@/types';
import { tokenStorage } from '@/utils/storage';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isBootstrapping: boolean;
  login: (payload: LoginRequest) => Promise<void>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>;
  logout: () => void;
  bootstrap: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: Boolean(tokenStorage.getAccessToken()),
  isBootstrapping: true,

  login: async (payload) => {
    const result = await authApi.login(payload);
    tokenStorage.set(result.access_token, result.refresh_token);
    set({ user: result.user, isAuthenticated: true, isBootstrapping: false });
  },

  /** Stores the FRESH pair the backend returns. Skipping that would sign the
   *  user out of THIS device too at the next refresh, because changing a password
   *  revokes every refresh token the account has. Other devices are meant to be
   *  signed out; the one doing the changing is not. */
  changePassword: async (currentPassword, newPassword) => {
    const pair = await authApi.changePassword(currentPassword, newPassword);
    tokenStorage.set(pair.access_token, pair.refresh_token);
  },

  /** Clears local state IMMEDIATELY, then tells the server.
   *
   *  Order matters: the user is signed out of this tab whatever the network
   *  does, and a failed call must never leave them stuck on a screen they think
   *  they left. But the call has to happen — without it the refresh token stays
   *  valid for its full TTL and the device keeps showing as active under
   *  /users/me/sessions, which is exactly what that screen is for. */
  logout: () => {
    const refreshToken = tokenStorage.getRefreshToken();
    tokenStorage.clear();
    set({ user: null, isAuthenticated: false, isBootstrapping: false });
    if (!refreshToken) return;
    // Fire-and-forget: revocation is best-effort from the client's side, and the
    // token is already gone locally either way.
    void authApi.logout(refreshToken).catch(() => undefined);
  },

  /** One-time app start: re-fetch user when a token exists. */
  bootstrap: async () => {
    if (!tokenStorage.getAccessToken()) {
      set({ user: null, isAuthenticated: false, isBootstrapping: false });
      return;
    }
    try {
      const user = await authApi.me();
      set({ user, isAuthenticated: true, isBootstrapping: false });
    } catch {
      tokenStorage.clear();
      set({ user: null, isAuthenticated: false, isBootstrapping: false });
    }
  },
}));

/** Overwrite `user` when another mutation (e.g. useUpdateProfile) returns a fresher record. Single place for this; the store never auto-syncs with React Query. */
export const syncAuthUser = (user: User) => useAuthStore.setState({ user });
