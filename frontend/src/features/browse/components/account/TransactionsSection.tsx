import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQueries } from '@tanstack/react-query';
import { accountApi } from '@/api/account.api';
import { ACCOUNT_QUERY_KEY } from '@/hooks/useAccount';
import { errorMessage } from '@/utils/error';
import { formatDateTime, formatVND } from '@/utils/format';
import type { Transaction } from '@/types';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import { INK_60, INK_65 } from '@/theme/customerTw';

const PAGE_SIZE = 10;

const STATUS_CLASS: Record<Transaction['status'], string> = {
  paid: 'bg-[#2e7d32]/22 text-[#2e7d32]',
  pending: 'bg-warning/22 text-warning',
  failed: 'bg-danger/20 text-danger',
  refunded: 'bg-info/20 text-info',
  refund_pending: 'bg-info/20 text-info',
};

/** Paginated money transaction list with a show-more pager. */
export const TransactionsSection = () => {
  const { t } = useTranslation();
  const [pageCount, setPageCount] = useState(1);

  const pages = useMemo(() => Array.from({ length: pageCount }, (_, i) => i + 1), [pageCount]);

  const results = useQueries({
    queries: pages.map((page) => ({
      queryKey: [ACCOUNT_QUERY_KEY, 'transactions', { page, page_size: PAGE_SIZE }],
      queryFn: () => accountApi.transactions({ page, page_size: PAGE_SIZE }),
    })),
  });

  const isLoading = results[0]?.isLoading ?? false;
  const isFetchingMore = results.length > 1 && (results[results.length - 1]?.isFetching ?? false);
  const error = results.find((r) => r.error)?.error;
  const lastData = results[results.length - 1]?.data;
  const hasMore = Boolean(lastData && lastData.meta.page < lastData.meta.total_pages);

  const shown = results.flatMap((r) => r.data?.items ?? []);

  return (
    <section>
      <h2 className="mt-0 mb-1 text-base font-bold">{t('customer.accountTransactions')}</h2>
      <p className={`mt-0 mb-4 text-[13px] ${INK_65}`}>{t('customer.accountTransactionsHint')}</p>

      {error ? (
        <Notice variant="error" className="mb-4">
          {errorMessage(error, t('common.somethingWrong'))}
        </Notice>
      ) : null}

      {isLoading ? <p className={INK_60}>{t('common.loading')}</p> : null}

      {!isLoading && shown.length === 0 && !error ? (
        <EmptyState>{t('customer.accountNoTransactions')}</EmptyState>
      ) : null}

      <div className="flex flex-col gap-2.5">
        {shown.map((tx) => (
          <div
            className="flex items-center justify-between gap-3 rounded-[10px] bg-[rgb(var(--cp-ink-rgb))]/5 px-3.5 py-3 max-[640px]:flex-col max-[640px]:items-start"
            key={tx.payment_id}
          >
            <div>
              <p className="m-0 truncate text-sm font-bold">{tx.movie_title}</p>
              <p className={`mt-0.75 text-xs ${INK_60}`}>
                {tx.hall_name} · {formatDateTime(tx.start_at)}
              </p>
              <p className={`mt-0.75 text-xs ${INK_60}`}>
                {t('booking.provider')}: {tx.provider} · {tx.txn_ref}
              </p>
            </div>
            <div className="shrink-0 text-right max-[640px]:text-left">
              <span className="block text-sm font-bold">
                {formatVND(tx.paid_amount ?? tx.amount)}
              </span>
              <span
                className={`mt-1 inline-block rounded-full px-2.5 py-0.5 text-[11px] font-bold ${STATUS_CLASS[tx.status]}`}
              >
                {t(`customer.accountTxStatus_${tx.status}`)}
              </span>
            </div>
          </div>
        ))}
      </div>

      {hasMore ? (
        <div className="mt-3.5 text-center">
          <Button
            variant="ghost"
            onClick={() => setPageCount((current) => current + 1)}
            disabled={isFetchingMore}
          >
            {isFetchingMore ? t('common.loading') : t('customer.loadMoreHistory')}
          </Button>
        </div>
      ) : null}
    </section>
  );
};

export default TransactionsSection;
