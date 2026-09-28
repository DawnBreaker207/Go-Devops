import type { PageQuery } from './api';

// Mirrors Go DTO internal/dto/article.go.

export type ArticleType = 'news' | 'promotion';
export type ArticleStatus = 'draft' | 'published' | 'hidden';

export interface Article {
  id: string;
  title: string;
  slug: string;
  summary?: string;
  thumbnail_url?: string;
  content: string;
  author_id: string;
  type: ArticleType;
  status: ArticleStatus;
  views: number;
  created_at: string;
  updated_at: string;
}

export interface ArticleListQuery extends PageQuery {
  type?: ArticleType;
}

export interface CreateArticlePayload {
  title: string;
  slug?: string;
  summary?: string;
  thumbnail_url?: string;
  content: string;
  type?: ArticleType;
  status?: ArticleStatus;
}

export interface UpdateArticlePayload {
  title?: string;
  slug?: string;
  summary?: string;
  thumbnail_url?: string;
  content?: string;
  type?: ArticleType;
  status?: ArticleStatus;
}
