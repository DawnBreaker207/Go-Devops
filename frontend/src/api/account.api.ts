import { apiClient, unwrap } from './client';
import { getDeviceId } from '@/utils/deviceId';
import type {
  ApiResponse,
  DeleteAccountPayload,
  NotificationPreference,
  PageQuery,
  Session,
  TransactionPage,
  UpdateNotificationPreferencePayload,
  UpdateProfilePayload,
  User,
} from '@/types';

export const accountApi = {
  updateProfile: (payload: UpdateProfilePayload) =>
    apiClient.put<ApiResponse<User>>('/users/me', payload).then(unwrap),

  deleteMe: (payload: DeleteAccountPayload) =>
    apiClient.delete<void>('/users/me', { data: payload }).then(() => undefined),

  transactions: (query: PageQuery) =>
    apiClient
      .get<ApiResponse<TransactionPage>>('/users/me/transactions', { params: query })
      .then(unwrap),

  getNotificationPreferences: () =>
    apiClient
      .get<ApiResponse<NotificationPreference>>('/users/me/notification-preferences')
      .then(unwrap),

  updateNotificationPreferences: (payload: UpdateNotificationPreferencePayload) =>
    apiClient
      .put<ApiResponse<NotificationPreference>>('/users/me/notification-preferences', payload)
      .then(unwrap),

  sessions: () =>
    apiClient
      .get<ApiResponse<Session[]>>('/users/me/sessions', { params: { device_id: getDeviceId() } })
      .then(unwrap),

  revokeSession: (id: string) =>
    apiClient.delete<ApiResponse<void>>(`/users/me/sessions/${id}`).then(() => undefined),
};
