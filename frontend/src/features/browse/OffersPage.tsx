import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import StaticPage from '@/components/ui/StaticPage';
import SectionHead from '@/components/ui/SectionHead';
import EmptyState from '@/components/ui/EmptyState';
import Notice from '@/components/ui/Notice';
import Panel from '@/components/ui/Panel';
import { useArticleList } from '@/features/article/hooks/useArticles';
import { usePublicCampaignList } from '@/features/campaign/hooks/useCampaigns';
import { offerDetailPath } from '@/routes/paths';
import { INK_60, INK_62 } from '@/theme/customerTw';
import { formatDateTime } from '@/utils/format';

const PAGE_SIZE = 20;

/** Promotions and news, backed by GET /articles (published only); replaces the old static empty state. */
export const OffersPage = () => {
  const { t } = useTranslation();
  const articles = useArticleList({ page: 1, page_size: PAGE_SIZE });
  // Active campaigns only, inside their own window (GET /campaigns is public and pre-filtered server-side).
  const campaigns = usePublicCampaignList({ page: 1, page_size: PAGE_SIZE });

  return (
    <StaticPage>
      <SectionHead title={t('customer.campaignsTitle')} />
      {campaigns.error ? (
        <Notice variant="error">{t('common.somethingWrong')}</Notice>
      ) : campaigns.isLoading ? (
        <p className={INK_62}>{t('common.loading')}</p>
      ) : !campaigns.data || campaigns.data.items.length === 0 ? (
        <EmptyState>{t('customer.campaignsEmpty')}</EmptyState>
      ) : (
        <div className="mb-8 grid gap-4">
          {campaigns.data.items.map((c) => (
            <Panel key={c.id}>
              <h3 className="m-0 mb-1 text-base font-bold text-white">{c.name}</h3>
              <p className={`m-0 mb-2 text-xs ${INK_60}`}>
                {t('customer.campaignWindow', {
                  start: formatDateTime(c.starts_at),
                  end: formatDateTime(c.ends_at),
                })}
              </p>
              {c.description ? (
                <p className={`m-0 mb-3 text-sm ${INK_60}`}>{c.description}</p>
              ) : null}

              {c.discount_codes.length > 0 ? (
                <div className="mb-3 flex flex-wrap gap-2">
                  {c.discount_codes.map((code) => (
                    <span
                      key={code.code}
                      className="rounded-md bg-brand px-2 py-1 font-mono text-xs font-bold text-on-brand"
                    >
                      {code.code} ·{' '}
                      {code.remaining !== undefined
                        ? t('customer.campaignRemaining', { count: code.remaining })
                        : t('customer.campaignRemainingUnlimited')}
                    </span>
                  ))}
                </div>
              ) : null}

              {c.combos.length > 0 ? (
                <div className="mb-3">
                  <p className={`m-0 mb-1 text-xs font-bold ${INK_60}`}>
                    {t('customer.campaignCombos')}
                  </p>
                  <ul className="m-0 list-disc pl-5 text-sm">
                    {c.combos.map((combo) => (
                      <li key={combo.combo_id}>{combo.name}</li>
                    ))}
                  </ul>
                </div>
              ) : null}

              {c.articles.length > 0 ? (
                <div>
                  <p className={`m-0 mb-1 text-xs font-bold ${INK_60}`}>
                    {t('customer.campaignArticles')}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {c.articles.map((article) => (
                      <Link
                        key={article.id}
                        to={offerDetailPath(article.slug)}
                        className="text-sm font-semibold text-brand no-underline hover:underline"
                      >
                        {article.title}
                      </Link>
                    ))}
                  </div>
                </div>
              ) : null}
            </Panel>
          ))}
        </div>
      )}

      <SectionHead title={t('customer.offersTitle')} />
      {articles.error ? (
        <Notice variant="error">{t('common.somethingWrong')}</Notice>
      ) : articles.isLoading ? (
        <p className={INK_62}>{t('common.loading')}</p>
      ) : !articles.data || articles.data.items.length === 0 ? (
        <EmptyState>{t('customer.offersEmpty')}</EmptyState>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {articles.data.items.map((a) => (
            <Link
              key={a.id}
              to={offerDetailPath(a.slug)}
              className="overflow-hidden rounded-2xl border border-white/10 bg-white/[0.04] no-underline transition-colors hover:border-white/25"
            >
              {a.thumbnail_url ? (
                <img
                  src={a.thumbnail_url}
                  alt=""
                  loading="lazy"
                  className="aspect-[16/9] w-full object-cover"
                />
              ) : null}
              <div className="p-4">
                <p className={`m-0 mb-1 text-xs ${INK_60}`}>
                  {t(`article.type${a.type === 'news' ? 'News' : 'Promotion'}`)} ·{' '}
                  {t('article.views', { count: a.views })}
                </p>
                <h3 className="m-0 mb-1 text-base font-bold text-white">{a.title}</h3>
                {a.summary ? <p className={`m-0 text-sm ${INK_60}`}>{a.summary}</p> : null}
              </div>
            </Link>
          ))}
        </div>
      )}
    </StaticPage>
  );
};

export default OffersPage;
