import { apiClient, unwrap } from './client';
import type { ApiResponse, UploadResult } from '@/types';

export const mediaApi = {
  uploadPoster: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return apiClient
      .post<ApiResponse<UploadResult>>('/admin/uploads/poster', form, {
        headers: { 'Content-Type': undefined },
      })
      .then(unwrap);
  },
};
