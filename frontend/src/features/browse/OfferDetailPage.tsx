import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import DOMPurify from 'dompurify';
import StaticPage from '@/components/ui/StaticPage';
import EmptyState from '@/components/ui/EmptyState';
import Notice from '@/components/ui/Notice';
import { useArticleDetail } from '@/features/article/hooks/useArticles';
import { PATHS } from '@/routes/paths';
import { formatDateTime } from '@/utils/format';
import { INK_60, INK_62 } from '@/theme/customerTw';

/** One published article, read by slug. Drafts/hidden answer 404 like an unknown slug. */
export const OfferDetailPage = () => {
  const { t } = useTranslation();
  const { slug } = useParams<{ slug: string }>();
  const article = useArticleDetail(slug ?? '');

  return (
    <StaticPage>
      <p className={`mt-0 mb-4 text-sm ${INK_60}`}>
        <Link to={PATHS.offers} className="text-brand no-underline">
          {t('customer.offersTitle')}
        </Link>
        {' / '}
        {t(`article.type${article.data?.type === 'promotion' ? 'Promotion' : 'News'}`)}
      </p>
      {article.error ? (
        <Notice variant="error">{t('common.somethingWrong')}</Notice>
      ) : article.isLoading ? (
        <p className={INK_62}>{t('common.loading')}</p>
      ) : !article.data ? (
        <EmptyState>{t('customer.offersEmpty')}</EmptyState>
      ) : (
        <article>
          <h1 className="mt-0 mb-2 text-2xl font-bold text-white">{article.data.title}</h1>
          <p className={`mt-0 mb-4 text-sm ${INK_60}`}>
            {formatDateTime(article.data.created_at)} ·{' '}
            {t('article.views', { count: article.data.views })}
          </p>
          {article.data.thumbnail_url ? (
            <img
              src={article.data.thumbnail_url}
              alt=""
              className="mb-4 aspect-[16/9] w-full rounded-2xl object-cover"
            />
          ) : null}
          {article.data.summary ? (
            <p className="mb-4 text-base font-semibold text-white">{article.data.summary}</p>
          ) : null}
          {/* Written with the admin's rich-text editor (Quill) as sanitized HTML, not plain text. */}
          <div
            className="text-[15px] leading-relaxed text-white/90"
            dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(article.data.content) }}
          />
        </article>
      )}
    </StaticPage>
  );
};

export default OfferDetailPage;
