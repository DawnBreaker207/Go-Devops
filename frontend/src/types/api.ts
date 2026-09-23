/** Response chuan tu BackEnd-CP: { code, message, data } */
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PageMeta {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface PagedData<T> {
  items: T[];
  meta: PageMeta;
}

export interface PageQuery {
  page?: number;
  page_size?: number;
  search?: string;
}

export interface ApiError {
  code: number;
  message: string;
  details?: Record<string, string>;
}
