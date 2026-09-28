import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { articleApi } from '@/api/article.api';
import type {
  ArticleListQuery,
  ArticleType,
  CreateArticlePayload,
  UpdateArticlePayload,
} from '@/types';

export const ARTICLE_QUERY_KEY = 'articles';
export const ARTICLE_ADMIN_QUERY_KEY = 'admin-articles';

const useInvalidateAdmin = () => {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: [ARTICLE_ADMIN_QUERY_KEY] });
    // A publish/hide changes what the public list shows too.
    queryClient.invalidateQueries({ queryKey: [ARTICLE_QUERY_KEY] });
  };
};

/* --- Public (published only) --- */

export const useArticleList = (query: ArticleListQuery) =>
  useQuery({
    queryKey: [ARTICLE_QUERY_KEY, query],
    queryFn: () => articleApi.list(query),
    placeholderData: (previous) => previous,
  });

export const useArticleDetail = (slug: string) =>
  useQuery({
    queryKey: [ARTICLE_QUERY_KEY, slug],
    queryFn: () => articleApi.detail(slug),
    enabled: slug.length > 0,
  });

/* --- Operator catalogue --- */

export const useAdminArticleList = (query: ArticleListQuery) =>
  useQuery({
    queryKey: [ARTICLE_ADMIN_QUERY_KEY, query],
    queryFn: () => articleApi.adminList(query),
    placeholderData: (previous) => previous,
  });

export const useCreateArticle = () => {
  const invalidate = useInvalidateAdmin();
  return useMutation({
    mutationFn: (payload: CreateArticlePayload) => articleApi.adminCreate(payload),
    onSuccess: () => void invalidate(),
  });
};

export const useUpdateArticle = () => {
  const invalidate = useInvalidateAdmin();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateArticlePayload }) =>
      articleApi.adminUpdate(id, payload),
    onSuccess: () => void invalidate(),
  });
};

export const useDeleteArticle = () => {
  const invalidate = useInvalidateAdmin();
  return useMutation({
    mutationFn: (id: string) => articleApi.adminDelete(id),
    onSuccess: () => void invalidate(),
  });
};

export type { ArticleType };
