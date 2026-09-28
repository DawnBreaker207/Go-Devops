import type { PageQuery } from './api';
import type { Article } from './article';
import type { DiscountCode } from './discount';

// Mirrors Go DTO internal/dto/campaign.go.
export interface Campaign {
  id: string;
  name: string;
  description?: string;
  starts_at: string;
  ends_at: string;
  active: boolean;
  per_user_limit: number;
  created_at: string;
  updated_at: string;
}

export interface CampaignListQuery extends PageQuery {
  active?: boolean;
}

export interface CreateCampaignPayload {
  name: string;
  description?: string;
  starts_at: string;
  ends_at: string;
  per_user_limit?: number;
  active?: boolean;
}

export interface UpdateCampaignPayload {
  name?: string;
  description?: string;
  starts_at?: string;
  ends_at?: string;
  per_user_limit?: number;
  active?: boolean;
}

export interface CampaignComboResponse {
  combo_id: string;
  name: string;
  price: number;
  promo_price?: number;
}

export interface AttachComboPayload {
  promo_price?: number;
}

export interface CampaignDetailResponse extends Campaign {
  discount_codes: DiscountCode[];
  combos: CampaignComboResponse[];
  articles: Article[];
}

export interface CampaignPublicDiscountCode {
  code: string;
  remaining?: number;
}

export interface CampaignPublicResponse {
  id: string;
  name: string;
  description?: string;
  starts_at: string;
  ends_at: string;
  discount_codes: CampaignPublicDiscountCode[];
  combos: CampaignComboResponse[];
  articles: Article[];
}
