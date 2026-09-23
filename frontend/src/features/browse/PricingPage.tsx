import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import SectionHead from '@/components/ui/SectionHead';
import Panel from '@/components/ui/Panel';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import { hallApi } from '@/api/hall.api';
import { SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';
import { INK_60, INK_65 } from '@/theme/customerTw';

/** Public price page, read from GET /pricing.
 *
 *  It used to be a hardcoded four-row table claiming weekday/weekend tiers. The
 *  backend has no such concept, and the numbers in it (75/90/110/190k) did not
 *  match a single row of `hall_prices` (70/100/160/130k) — so the page told
 *  customers a price they would never be charged. Prices are per HALL and per
 *  seat type, which is why this renders one block per hall. */
export const PricingPage = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ['public-pricing'],
    queryFn: () => hallApi.publicPrices(),
    staleTime: 5 * 60_000,
  });

  const halls = data?.halls ?? [];

  return (
    <>
      <SectionHead title={t('customer.navPricing')} subtitle={t('customer.pricingSubtitle')} />

      {error ? (
        <Notice variant="error">{errorMessage(error, t('common.somethingWrong'))}</Notice>
      ) : null}

      {isLoading ? <p className={INK_60}>{t('common.loading')}</p> : null}

      {!isLoading && !error && halls.length === 0 ? (
        <EmptyState>{t('customer.pricingEmpty')}</EmptyState>
      ) : null}

      {data && data.from_price > 0 ? (
        <p className={`mx-auto mb-5 max-w-160 text-center text-lg font-bold`}>
          {t('customer.pricingFrom', { price: formatVND(data.from_price) })}
        </p>
      ) : null}

      {halls.map((hall) => (
        <Panel key={hall.hall_id} className="mx-auto mb-4 max-w-160">
          <h2 className="mt-0 mb-3 text-base font-bold">{hall.hall_name}</h2>
          <div className="divide-y divide-white/10">
            {/* SEAT_TYPES, not Object.keys: the map's key order is not guaranteed
                and every other screen shows the four types in this same order. */}
            {SEAT_TYPES.map((seatType) => (
              <div key={seatType} className="flex items-center justify-between py-3">
                <span>{t(`hall.seatType_${seatType}`)}</span>
                <span className="font-bold tabular-nums">{formatVND(hall.prices[seatType])}</span>
              </div>
            ))}
          </div>
        </Panel>
      ))}

      {halls.length > 0 ? (
        <p className={`mx-auto mt-4 max-w-160 text-center text-xs ${INK_65}`}>
          {t('customer.pricingNote')}
        </p>
      ) : null}
    </>
  );
};

export default PricingPage;
