import { Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import DiscountsPage from '@/features/discount/DiscountsPage';
import CampaignsPage from '@/features/campaign/CampaignsPage';
import ArticlesPage from '@/features/article/ArticlesPage';

/** Promotion group: Discounts/Campaigns/Articles tabbed under one sider entry (ROLES_ADMIN).
 *  Grouped together because a Campaign can attach both a discount code and an article
 *  (backend `attach_campaign_article`/`attach_campaign_discount`) - all three are the same
 *  promotional-content workflow, unlike Pricing which is a separate base-price engine. */
export const PromotionPage = () => {
  const { t } = useTranslation();

  return (
    <>
      <PageHeader title={t('menu.promotions')} />
      <Tabs
        defaultActiveKey="discounts"
        items={[
          { key: 'discounts', label: t('menu.discounts'), children: <DiscountsPage /> },
          { key: 'campaigns', label: t('menu.campaigns'), children: <CampaignsPage /> },
          { key: 'articles', label: t('menu.articles'), children: <ArticlesPage /> },
        ]}
      />
    </>
  );
};

export default PromotionPage;
