import { apiClient, unwrap } from './client';
import type { ApiResponse, UploadResult } from '@/types';

export const mediaApi = {
  /** POST /admin/uploads/poster — multipart, field name MUST be "file".
   *
   *  Operator scope (admin AND staff).
   *
   *  `'Content-Type': undefined` is load-bearing. The shared client is created
   *  with `headers: { 'Content-Type': 'application/json' }`, and with that in
   *  place axios does NOT detect the FormData — it JSON-stringifies it and sends
   *  `{"file":{"uid":"rc-upload-..."}}`, which the backend rejects with
   *  400 `multipart field "file" is required`. Clearing the header lets axios and
   *  the browser set `multipart/form-data` WITH the boundary, which is the part
   *  that cannot be written by hand. */
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
