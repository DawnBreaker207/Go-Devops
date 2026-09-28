import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import SectionHead from '@/components/ui/SectionHead';
import Panel from '@/components/ui/Panel';
import Notice from '@/components/ui/Notice';
import EmptyState from '@/components/ui/EmptyState';
import { pricingApi } from '@/api/pricing.api';
import { SEAT_TYPES } from '@/types';
import { errorMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';
import { INK_60, INK_65 } from '@/theme/customerTw';

/** Public price page, read from GET /pricing. Global pricing redesign (PLAN_CAMPAIGN.md section
 *  11.4, Phase 3): prices are no longer per-hall, so this renders ONE panel with the 4 seat
 *  types' single global price instead of one panel per hall. */
export const PricingPage = () => {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ['public-pricing'],
    queryFn: () => pricingApi.publicPrices(),
    staleTime: 5 * 60_000,
  });

  // "Empty" used to mean "no halls have prices set"; now there is only one global block, so
  // empty means no base price has been configured at all (from_price === 0, per
  // dto.PublicPriceListResponse: "0 when nothing is configured yet").
  const isEmpty = data !== undefined && data.from_price <= 0;

  return (
    <>
      <SectionHead title={t('customer.navPricing')} subtitle={t('customer.pricingSubtitle')} />

      {error ? (
        <Notice variant="error">{errorMessage(error, t('common.somethingWrong'))}</Notice>
      ) : null}

      {isLoading ? <p className={INK_60}>{t('common.loading')}</p> : null}

      {!isLoading && !error && isEmpty ? (
        <EmptyState>{t('customer.pricingEmpty')}</EmptyState>
      ) : null}

      {data && data.from_price > 0 ? (
        <p className={`mx-auto mb-5 max-w-160 text-center text-lg font-bold`}>
          {t('customer.pricingFrom', { price: formatVND(data.from_price) })}
        </p>
      ) : null}

      {data && !isEmpty ? (
        <Panel className="mx-auto mb-4 max-w-160">
          <div className="divide-y divide-white/10">
            {/* SEAT_TYPES, not Object.keys: the map's key order is not guaranteed
                and every other screen shows the four types in this same order. */}
            {SEAT_TYPES.map((seatType) => (
              <div key={seatType} className="flex items-center justify-between py-3">
                <span>{t(`hall.seatType_${seatType}`)}</span>
                <span className="font-bold tabular-nums">{formatVND(data.prices[seatType])}</span>
              </div>
            ))}
          </div>
        </Panel>
      ) : null}

      {data && !isEmpty ? (
        <p className={`mx-auto mt-4 max-w-160 text-center text-xs ${INK_65}`}>
          {t('customer.pricingNote')}
        </p>
      ) : null}
    </>
  );
};

export default PricingPage;
