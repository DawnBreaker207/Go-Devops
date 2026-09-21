import { useTranslation } from 'react-i18next';
import SectionHead from '@/components/ui/SectionHead';
import Panel from '@/components/ui/Panel';
import { INK_65 } from '@/theme/customerTw';

interface PriceRow {
  labelKey: string;
  price: string;
}

/** Static reference table, no backend (no per-category pricing model exists, only per-seat-type hall prices). Exact per-show/seat prices still come from the booking API. */
const ROWS: PriceRow[] = [
  { labelKey: 'pricingStandardWeekday', price: '75.000đ' },
  { labelKey: 'pricingStandardWeekend', price: '90.000đ' },
  { labelKey: 'pricingVip', price: '110.000đ' },
  { labelKey: 'pricingCouple', price: '190.000đ' },
];

export const PricingPage = () => {
  const { t } = useTranslation();

  return (
    <>
      <SectionHead title={t('customer.navPricing')} subtitle={t('customer.pricingSubtitle')} />

      <Panel className="mx-auto max-w-160">
        <div className="divide-y divide-white/10">
          {ROWS.map((row) => (
            <div key={row.labelKey} className="flex items-center justify-between py-3">
              <span>{t(`customer.${row.labelKey}`)}</span>
              <span className="font-bold">{row.price}</span>
            </div>
          ))}
        </div>
        <p className={`mt-4 text-xs ${INK_65}`}>{t('customer.pricingNote')}</p>
      </Panel>
    </>
  );
};

export default PricingPage;
