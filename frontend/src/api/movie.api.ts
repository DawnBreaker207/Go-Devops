import { apiClient, unwrap } from './client';
import type { ApiResponse, Movie, MoviePayload, PageQuery, PagedData } from '@/types';

export const movieApi = {
  list: (query: PageQuery) =>
    apiClient.get<ApiResponse<PagedData<Movie>>>('/movies', { params: query }).then(unwrap),

  detail: (id: string) => apiClient.get<ApiResponse<Movie>>(`/movies/${id}`).then(unwrap),

  create: (payload: MoviePayload) =>
    apiClient.post<ApiResponse<Movie>>('/movies', payload).then(unwrap),

  update: (id: string, payload: MoviePayload) =>
    apiClient.put<ApiResponse<Movie>>(`/movies/${id}`, payload).then(unwrap),

  remove: (id: string) => apiClient.delete<ApiResponse<null>>(`/movies/${id}`).then(unwrap),
};
