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
  /** Full replace (both fields required); email is immutable server-side. */
  updateProfile: (payload: UpdateProfilePayload) =>
    apiClient.put<ApiResponse<User>>('/users/me', payload).then(unwrap),

  /** 204 empty body, never unwrap. Wrong password is 401; open CONFIRMED tickets block with 409. */
  deleteMe: (payload: DeleteAccountPayload) =>
    apiClient.delete<void>('/users/me', { data: payload }).then(() => undefined),

  /** Paginated finance view (unlike the ticket-oriented GET /orders). */
  transactions: (query: PageQuery) =>
    apiClient
      .get<ApiResponse<TransactionPage>>('/users/me/transactions', { params: query })
      .then(unwrap),

  /** First read creates the default row (both on) when none exists. */
  getNotificationPreferences: () =>
    apiClient
      .get<ApiResponse<NotificationPreference>>('/users/me/notification-preferences')
      .then(unwrap),

  /** Replaces both flags at once; both false is valid. */
  updateNotificationPreferences: (payload: UpdateNotificationPreferencePayload) =>
    apiClient
      .put<ApiResponse<NotificationPreference>>('/users/me/notification-preferences', payload)
      .then(unwrap),

  /** Bare array, unpaginated. Sends this browser's device_id to mark `is_current`. */
  sessions: () =>
    apiClient
      .get<ApiResponse<Session[]>>('/users/me/sessions', { params: { device_id: getDeviceId() } })
      .then(unwrap),

  /** 200 with envelope but no data; signs out one device, others unaffected. */
  revokeSession: (id: string) =>
    apiClient.delete<ApiResponse<void>>(`/users/me/sessions/${id}`).then(() => undefined),
};
