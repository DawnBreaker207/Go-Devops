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

  changePassword: async (currentPassword, newPassword) => {
    const pair = await authApi.changePassword(currentPassword, newPassword);
    tokenStorage.set(pair.access_token, pair.refresh_token);
  },

  logout: () => {
    const refreshToken = tokenStorage.getRefreshToken();
    tokenStorage.clear();
    set({ user: null, isAuthenticated: false, isBootstrapping: false });
    if (!refreshToken) return;
    void authApi.logout(refreshToken).catch(() => undefined);
  },

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

export const syncAuthUser = (user: User) => useAuthStore.setState({ user });
