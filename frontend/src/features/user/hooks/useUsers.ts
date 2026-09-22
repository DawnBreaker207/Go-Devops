import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { userApi } from '@/api/user.api';
import { DASHBOARD_QUERY_KEY } from '@/features/dashboard/hooks/useDashboard';
import type { CreateUserPayload, UpdateUserPayload, UserListQuery } from '@/types';

export const USER_QUERY_KEY = 'users';

export const useUserList = (query: UserListQuery) =>
  useQuery({
    queryKey: [USER_QUERY_KEY, query],
    queryFn: () => userApi.list(query),
    placeholderData: (previous) => previous,
  });

/** GET /admin/stats counts locked accounts too, so only CREATING changes it -
 *  locking or re-roling leaves the total untouched. */
const invalidateStats = (queryClient: ReturnType<typeof useQueryClient>) =>
  queryClient.invalidateQueries({ queryKey: [DASHBOARD_QUERY_KEY] });

export const useCreateUser = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateUserPayload) => userApi.create(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [USER_QUERY_KEY] });
      void invalidateStats(queryClient);
    },
  });
};

export const useUpdateUser = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateUserPayload }) =>
      userApi.update(id, payload),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [USER_QUERY_KEY] }),
  });
};
