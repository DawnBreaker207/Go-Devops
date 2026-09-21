import { create } from 'zustand';
import { authApi } from '@/api/auth.api';
import type { LoginRequest, User } from '@/types';
import { tokenStorage } from '@/utils/storage';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isBootstrapping: boolean;
  login: (payload: LoginRequest) => Promise<void>;
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

  logout: () => {
    tokenStorage.clear();
    set({ user: null, isAuthenticated: false, isBootstrapping: false });
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
