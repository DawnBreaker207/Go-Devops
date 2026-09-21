import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { accountApi } from '@/api/account.api';
import { syncAuthUser } from '@/stores/authStore';
import type {
  DeleteAccountPayload,
  PageQuery,
  UpdateNotificationPreferencePayload,
  UpdateProfilePayload,
} from '@/types';

/** GET/PUT /users/me* is shared by every authenticated role (see router.go `protected` group), so this hook lives here; also reused by ProfilePage. Exception: useDeleteAccount is customer-only per RequireRoles. */
export const ACCOUNT_QUERY_KEY = 'account';

/** authStore.user is stale after this mutation; syncAuthUser keeps header/avatar current. */
export const useUpdateProfile = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateProfilePayload) => accountApi.updateProfile(payload),
    onSuccess: (user) => {
      syncAuthUser(user);
      queryClient.invalidateQueries({ queryKey: [ACCOUNT_QUERY_KEY] });
    },
  });
};

export const useTransactions = (query: PageQuery) =>
  useQuery({
    queryKey: [ACCOUNT_QUERY_KEY, 'transactions', query],
    queryFn: () => accountApi.transactions(query),
    placeholderData: (previous) => previous,
  });

export const useNotificationPreferences = () =>
  useQuery({
    queryKey: [ACCOUNT_QUERY_KEY, 'notification-preferences'],
    queryFn: () => accountApi.getNotificationPreferences(),
  });

export const useUpdateNotificationPreferences = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateNotificationPreferencePayload) =>
      accountApi.updateNotificationPreferences(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [ACCOUNT_QUERY_KEY, 'notification-preferences'] });
    },
  });
};

export const useSessions = () =>
  useQuery({
    queryKey: [ACCOUNT_QUERY_KEY, 'sessions'],
    queryFn: () => accountApi.sessions(),
  });

export const useRevokeSession = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => accountApi.revokeSession(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [ACCOUNT_QUERY_KEY, 'sessions'] });
    },
  });
};

/** No invalidation: caller logs out and routes home, dropping all user cache. */
export const useDeleteAccount = () =>
  useMutation({
    mutationFn: (payload: DeleteAccountPayload) => accountApi.deleteMe(payload),
  });
