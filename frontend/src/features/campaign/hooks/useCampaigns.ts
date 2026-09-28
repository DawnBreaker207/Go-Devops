import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { campaignApi } from '@/api/campaign.api';
import type {
  AttachComboPayload,
  CampaignListQuery,
  CreateCampaignPayload,
  PageQuery,
  UpdateCampaignPayload,
} from '@/types';

export const CAMPAIGN_QUERY_KEY = 'campaigns';

export const useCampaignList = (query: CampaignListQuery) =>
  useQuery({
    queryKey: [CAMPAIGN_QUERY_KEY, query],
    queryFn: () => campaignApi.adminList(query),
    placeholderData: (previous) => previous,
  });

export const useCampaignDetail = (id: string | null) =>
  useQuery({
    queryKey: [CAMPAIGN_QUERY_KEY, id],
    queryFn: () => campaignApi.adminGet(id as string),
    enabled: Boolean(id),
  });

/* --- Public: only active campaigns inside their window --- */

export const usePublicCampaignList = (query: PageQuery) =>
  useQuery({
    queryKey: [CAMPAIGN_QUERY_KEY, 'public', query],
    queryFn: () => campaignApi.publicList(query),
    placeholderData: (previous) => previous,
  });

const useInvalidate = () => {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: [CAMPAIGN_QUERY_KEY] });
};

export const useCreateCampaign = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (payload: CreateCampaignPayload) => campaignApi.adminCreate(payload),
    onSuccess: () => void invalidate(),
  });
};

export const useUpdateCampaign = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateCampaignPayload }) =>
      campaignApi.adminUpdate(id, payload),
    onSuccess: () => void invalidate(),
  });
};

export const useDeleteCampaign = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (id: string) => campaignApi.adminDelete(id),
    onSuccess: () => void invalidate(),
  });
};

/* --- Attach / detach existing catalogue entities --- */

export const useAttachCampaignCombo = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({
      id,
      comboId,
      payload,
    }: {
      id: string;
      comboId: string;
      payload?: AttachComboPayload;
    }) => campaignApi.attachCombo(id, comboId, payload),
    onSuccess: () => void invalidate(),
  });
};

export const useDetachCampaignCombo = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, comboId }: { id: string; comboId: string }) =>
      campaignApi.detachCombo(id, comboId),
    onSuccess: () => void invalidate(),
  });
};

export const useAttachCampaignArticle = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, articleId }: { id: string; articleId: string }) =>
      campaignApi.attachArticle(id, articleId),
    onSuccess: () => void invalidate(),
  });
};

export const useDetachCampaignArticle = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, articleId }: { id: string; articleId: string }) =>
      campaignApi.detachArticle(id, articleId),
    onSuccess: () => void invalidate(),
  });
};

export const useAttachCampaignDiscountCode = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, codeId }: { id: string; codeId: string }) =>
      campaignApi.attachDiscountCode(id, codeId),
    onSuccess: () => void invalidate(),
  });
};

export const useDetachCampaignDiscountCode = () => {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, codeId }: { id: string; codeId: string }) =>
      campaignApi.detachDiscountCode(id, codeId),
    onSuccess: () => void invalidate(),
  });
};
