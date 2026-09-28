import { useState } from 'react';
import { App, Button, Empty, Modal, Select, Space, Tag, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import { useDiscountList } from '@/features/discount/hooks/useDiscounts';
import { useConcessionList } from '@/features/concession/hooks/useConcessions';
import { useAdminArticleList } from '@/features/article/hooks/useArticles';
import {
  useAttachCampaignArticle,
  useAttachCampaignCombo,
  useAttachCampaignDiscountCode,
  useCampaignDetail,
  useDetachCampaignArticle,
  useDetachCampaignCombo,
  useDetachCampaignDiscountCode,
} from '../hooks/useCampaigns';
import { errorMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';

interface CampaignLinksModalProps {
  open: boolean;
  /** null while nothing is selected; the modal is only ever opened with an id. */
  campaignId: string | null;
  onCancel: () => void;
}

/** v1 "record/display only" attach UI: pick an existing discount code / combo /
 *  article by id and link it to the campaign. No creation here - the catalogue
 *  pages own that. Mirrors DiscountsPage/ConcessionsPage/ArticlesPage's own
 *  list hooks rather than a new "picker" component, since nothing reusable for
 *  this existed yet (TODO: confirm if a shared entity-picker is wanted later). */
export const CampaignLinksModal = ({ open, campaignId, onCancel }: CampaignLinksModalProps) => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const [pickedCode, setPickedCode] = useState<string | null>(null);
  const [pickedCombo, setPickedCombo] = useState<string | null>(null);
  const [pickedArticle, setPickedArticle] = useState<string | null>(null);

  const detail = useCampaignDetail(campaignId);

  // Wide page size: catalogue lists are small operator-owned data, a v1 Select is enough.
  const discounts = useDiscountList({ page: 1, page_size: 100, active: true });
  const combos = useConcessionList({ page: 1, page_size: 100, active: true });
  const articles = useAdminArticleList({ page: 1, page_size: 100 });

  const attachCombo = useAttachCampaignCombo();
  const detachCombo = useDetachCampaignCombo();
  const attachArticle = useAttachCampaignArticle();
  const detachArticle = useDetachCampaignArticle();
  const attachCode = useAttachCampaignDiscountCode();
  const detachCode = useDetachCampaignDiscountCode();

  if (!campaignId) return null;

  const linkedCodeIds = new Set(detail.data?.discount_codes.map((c) => c.id) ?? []);
  const linkedComboIds = new Set(detail.data?.combos.map((c) => c.combo_id) ?? []);
  const linkedArticleIds = new Set(detail.data?.articles.map((a) => a.id) ?? []);

  const codeOptions = (discounts.data?.items ?? [])
    .filter((c) => !linkedCodeIds.has(c.id))
    .map((c) => ({ value: c.id, label: c.code }));
  const comboOptions = (combos.data?.items ?? [])
    .filter((c) => !linkedComboIds.has(c.id))
    .map((c) => ({ value: c.id, label: `${c.name} (${formatVND(c.price)})` }));
  const articleOptions = (articles.data?.items ?? [])
    .filter((a) => !linkedArticleIds.has(a.id))
    .map((a) => ({ value: a.id, label: a.title }));

  const handleAttachCode = async () => {
    if (!pickedCode) return;
    try {
      await attachCode.mutateAsync({ id: campaignId, codeId: pickedCode });
      setPickedCode(null);
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleAttachCombo = async () => {
    if (!pickedCombo) return;
    try {
      await attachCombo.mutateAsync({ id: campaignId, comboId: pickedCombo });
      setPickedCombo(null);
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleAttachArticle = async () => {
    if (!pickedArticle) return;
    try {
      await attachArticle.mutateAsync({ id: campaignId, articleId: pickedArticle });
      setPickedArticle(null);
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleDetachCode = async (codeId: string) => {
    try {
      await detachCode.mutateAsync({ id: campaignId, codeId });
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleDetachCombo = async (comboId: string) => {
    try {
      await detachCombo.mutateAsync({ id: campaignId, comboId });
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  const handleDetachArticle = async (articleId: string) => {
    try {
      await detachArticle.mutateAsync({ id: campaignId, articleId });
    } catch (error) {
      message.error(errorMessage(error, t('common.somethingWrong')));
    }
  };

  return (
    <Modal
      open={open}
      title={t('campaign.linksTitle')}
      footer={null}
      onCancel={onCancel}
      destroyOnHidden
      width={620}
    >
      <Typography.Title level={5}>{t('campaign.linkedCodes')}</Typography.Title>
      <Space wrap style={{ marginBottom: 8 }}>
        {detail.data?.discount_codes.length ? (
          detail.data.discount_codes.map((c) => (
            <Tag
              key={c.id}
              closable
              onClose={(e) => {
                e.preventDefault();
                void handleDetachCode(c.id);
              }}
            >
              {c.code}
            </Tag>
          ))
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('campaign.noneLinked')} />
        )}
      </Space>
      <Space.Compact style={{ width: '100%', marginBottom: 20 }}>
        <Select
          showSearch
          placeholder={t('campaign.pickCode')}
          style={{ width: '100%' }}
          value={pickedCode}
          options={codeOptions}
          filterOption={(input, option) =>
            (option?.label ?? '').toString().toLowerCase().includes(input.toLowerCase())
          }
          onChange={setPickedCode}
        />
        <Button
          type="primary"
          disabled={!pickedCode}
          loading={attachCode.isPending}
          onClick={() => void handleAttachCode()}
        >
          {t('campaign.attach')}
        </Button>
      </Space.Compact>

      <Typography.Title level={5}>{t('campaign.linkedCombos')}</Typography.Title>
      <Space wrap style={{ marginBottom: 8 }}>
        {detail.data?.combos.length ? (
          detail.data.combos.map((c) => (
            <Tag
              key={c.combo_id}
              closable
              onClose={(e) => {
                e.preventDefault();
                void handleDetachCombo(c.combo_id);
              }}
            >
              {c.name}
            </Tag>
          ))
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('campaign.noneLinked')} />
        )}
      </Space>
      <Space.Compact style={{ width: '100%', marginBottom: 20 }}>
        <Select
          showSearch
          placeholder={t('campaign.pickCombo')}
          style={{ width: '100%' }}
          value={pickedCombo}
          options={comboOptions}
          filterOption={(input, option) =>
            (option?.label ?? '').toString().toLowerCase().includes(input.toLowerCase())
          }
          onChange={setPickedCombo}
        />
        <Button
          type="primary"
          disabled={!pickedCombo}
          loading={attachCombo.isPending}
          onClick={() => void handleAttachCombo()}
        >
          {t('campaign.attach')}
        </Button>
      </Space.Compact>

      <Typography.Title level={5}>{t('campaign.linkedArticles')}</Typography.Title>
      <Space wrap style={{ marginBottom: 8 }}>
        {detail.data?.articles.length ? (
          detail.data.articles.map((a) => (
            <Tag
              key={a.id}
              closable
              onClose={(e) => {
                e.preventDefault();
                void handleDetachArticle(a.id);
              }}
            >
              {a.title}
            </Tag>
          ))
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('campaign.noneLinked')} />
        )}
      </Space>
      <Space.Compact style={{ width: '100%' }}>
        <Select
          showSearch
          placeholder={t('campaign.pickArticle')}
          style={{ width: '100%' }}
          value={pickedArticle}
          options={articleOptions}
          filterOption={(input, option) =>
            (option?.label ?? '').toString().toLowerCase().includes(input.toLowerCase())
          }
          onChange={setPickedArticle}
        />
        <Button
          type="primary"
          disabled={!pickedArticle}
          loading={attachArticle.isPending}
          onClick={() => void handleAttachArticle()}
        >
          {t('campaign.attach')}
        </Button>
      </Space.Compact>
    </Modal>
  );
};

export default CampaignLinksModal;
