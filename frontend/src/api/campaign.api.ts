import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  AttachComboPayload,
  Campaign,
  CampaignComboResponse,
  CampaignDetailResponse,
  CampaignListQuery,
  CampaignPublicResponse,
  CreateCampaignPayload,
  DiscountCode,
  PagedData,
  PageQuery,
  UpdateCampaignPayload,
  Article,
} from '@/types';

export const campaignApi = {
  publicList: (query: PageQuery) =>
    apiClient
      .get<ApiResponse<PagedData<CampaignPublicResponse>>>('/campaigns', { params: query })
      .then(unwrap),

  publicGet: (id: string) =>
    apiClient.get<ApiResponse<CampaignPublicResponse>>(`/campaigns/${id}`).then(unwrap),

  adminList: (query: CampaignListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<Campaign>>>('/admin/campaigns', { params: query })
      .then(unwrap),

  adminGet: (id: string) =>
    apiClient.get<ApiResponse<CampaignDetailResponse>>(`/admin/campaigns/${id}`).then(unwrap),

  adminCreate: (payload: CreateCampaignPayload) =>
    apiClient.post<ApiResponse<Campaign>>('/admin/campaigns', payload).then(unwrap),

  adminUpdate: (id: string, payload: UpdateCampaignPayload) =>
    apiClient.patch<ApiResponse<Campaign>>(`/admin/campaigns/${id}`, payload).then(unwrap),

  adminDelete: (id: string) =>
    apiClient.delete<void>(`/admin/campaigns/${id}`).then(() => undefined),

  attachCombo: (id: string, comboId: string, payload?: AttachComboPayload) =>
    apiClient
      .post<ApiResponse<CampaignComboResponse>>(
        `/admin/campaigns/${id}/combos/${comboId}`,
        payload ?? {}
      )
      .then(unwrap),

  detachCombo: (id: string, comboId: string) =>
    apiClient.delete<void>(`/admin/campaigns/${id}/combos/${comboId}`).then(() => undefined),

  attachArticle: (id: string, articleId: string) =>
    apiClient
      .post<ApiResponse<Article>>(`/admin/campaigns/${id}/articles/${articleId}`, {})
      .then(unwrap),

  detachArticle: (id: string, articleId: string) =>
    apiClient.delete<void>(`/admin/campaigns/${id}/articles/${articleId}`).then(() => undefined),

  attachDiscountCode: (id: string, codeId: string) =>
    apiClient
      .post<ApiResponse<DiscountCode>>(`/admin/campaigns/${id}/discount-codes/${codeId}`, {})
      .then(unwrap),

  detachDiscountCode: (id: string, codeId: string) =>
    apiClient.delete<void>(`/admin/campaigns/${id}/discount-codes/${codeId}`).then(() => undefined),
};
