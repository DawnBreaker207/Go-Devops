import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { accountApi } from '@/api/account.api';
import { syncAuthUser } from '@/stores/authStore';
import type {
  DeleteAccountPayload,
  PageQuery,
  UpdateNotificationPreferencePayload,
  UpdateProfilePayload,
} from '@/types';

export const ACCOUNT_QUERY_KEY = 'account';

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

export const useDeleteAccount = () =>
  useMutation({
    mutationFn: (payload: DeleteAccountPayload) => accountApi.deleteMe(payload),
  });
