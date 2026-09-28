import { apiClient, unwrap } from './client';
import type {
  ApiResponse,
  Article,
  ArticleListQuery,
  CreateArticlePayload,
  PagedData,
  UpdateArticlePayload,
} from '@/types';

export const articleApi = {
  list: (query: ArticleListQuery) =>
    apiClient.get<ApiResponse<PagedData<Article>>>('/articles', { params: query }).then(unwrap),

  detail: (slug: string) => apiClient.get<ApiResponse<Article>>(`/articles/${slug}`).then(unwrap),

  adminList: (query: ArticleListQuery) =>
    apiClient
      .get<ApiResponse<PagedData<Article>>>('/admin/articles', { params: query })
      .then(unwrap),

  adminGet: (id: string) =>
    apiClient.get<ApiResponse<Article>>(`/admin/articles/${id}`).then(unwrap),

  adminCreate: (payload: CreateArticlePayload) =>
    apiClient.post<ApiResponse<Article>>('/admin/articles', payload).then(unwrap),

  adminUpdate: (id: string, payload: UpdateArticlePayload) =>
    apiClient.put<ApiResponse<Article>>(`/admin/articles/${id}`, payload).then(unwrap),

  adminDelete: (id: string) =>
    apiClient.delete<void>(`/admin/articles/${id}`).then(() => undefined),
};
