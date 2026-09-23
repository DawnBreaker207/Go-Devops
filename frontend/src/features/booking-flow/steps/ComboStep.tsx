import { useTranslation } from 'react-i18next';
import type { Combo, SeatType } from '@/types';
import { useBookingFlowStore } from '@/stores/bookingFlowStore';
import { useAuthStore } from '@/stores/authStore';
import { formatVND } from '@/utils/format';
import Notice from '@/components/ui/Notice';
import {
  INK,
  INK_60,
  INK_62,
  INK_65,
  INK_BG_035,
  INK_BG_06,
  INK_BG_08,
  INK_BORDER_14,
  INK_BORDER_28,
} from '@/theme/customerTw';

// Like buttonStyles.ts: no Tailwind variant for `:hover:not(:disabled)` -
// approximate (disabled already dims + unclickable).
const STEPPER_BTN_CLASS =
  `inline-flex h-8 w-8 items-center justify-center rounded-lg border bg-transparent ` +
  `text-base leading-none font-bold cursor-pointer transition-[transform,border-color] ${INK_BORDER_28} ${INK} ` +
  'duration-fast ease-out hover-fine:border-brand active:scale-[var(--motion-scale-press)] ' +
  'disabled:cursor-not-allowed disabled:opacity-35 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand';

interface ComboStepProps {
  combos: Combo[];
  combosLoading: boolean;
  qty: Record<string, number>;
  maxQty: number;
  comboError: string | null;
  onQty: (comboId: string, delta: number) => void;
}

/** Step 2: order recap + snacks. Qty/order state lives in the parent; Skip/Continue are sidebar buttons. */
export const ComboStep = ({
  combos,
  combosLoading,
  qty,
  maxQty,
  comboError,
  onQty,
}: ComboStepProps) => {
  const { t } = useTranslation();
  const hasCombos = combos.length > 0;

  // Booker from the signed-in account (read-only here).
  const user = useAuthStore((s) => s.user);
  // Seats from the shared store (survives F5) - grouped by kind + price.
  const flowSeats = useBookingFlowStore((s) => s.seats);
  const seatGroups = (() => {
    const map = new Map<string, { seatType: SeatType; price: number; labels: string[] }>();
    for (const s of flowSeats) {
      const key = `${s.seatType}@${s.price}`;
      const g = map.get(key) ?? { seatType: s.seatType, price: s.price, labels: [] };
      g.labels.push(s.label);
      map.set(key, g);
    }
    return [...map.values()];
  })();

  return (
    <>
      {/* Read-only payer info (name/email/phone) - bare, ABOVE the combo title. */}
      {user ? (
        <section className="mb-6">
          <h2
            className={`mt-0 mb-2.5 text-[28px] font-bold tracking-[1px] uppercase max-[640px]:text-[22px] ${INK_65}`}
          >
            {t('customer.paymentInfoTitle')}
          </h2>
          <div className="flex justify-between gap-4 py-1.5 text-[15px]">
            <span className={INK_60}>{t('user.fullName')}</span>
            <span className="text-right font-semibold">{user.full_name}</span>
          </div>
          <div className="flex justify-between gap-4 py-1.5 text-[15px]">
            <span className={INK_60}>{t('user.email')}</span>
            <span className="text-right font-semibold">{user.email}</span>
          </div>
          {user.phone ? (
            <div className="flex justify-between gap-4 py-1.5 text-[15px]">
              <span className={INK_60}>{t('customer.accountPhone')}</span>
              <span className="text-right font-semibold">{user.phone}</span>
            </div>
          ) : null}
          {seatGroups.length > 0 ? (
            <>
              <div className={`my-2.5 h-px ${INK_BG_08}`} />
              {seatGroups.map((g) => (
                <div
                  key={`${g.seatType}@${g.price}`}
                  className="flex justify-between gap-4 py-1.5 text-[15px]"
                >
                  <span className={INK_60}>{t(`booking.seatType_${g.seatType}`)}</span>
                  <span className="text-right tabular-nums">
                    <span className={`font-medium ${INK_60}`}>
                      {formatVND(g.price)} × {g.labels.length} =
                    </span>{' '}
                    <span className="font-semibold">{formatVND(g.price * g.labels.length)}</span>
                  </span>
                </div>
              ))}
            </>
          ) : null}
        </section>
      ) : null}

      <h1
        className={`mt-6 mb-2.5 text-[28px] font-bold tracking-[1px] uppercase max-[640px]:text-[22px] ${INK_65}`}
      >
        {t('customer.stepCombo')}
      </h1>
      <p className={`-mt-2 mb-5 ${INK_62}`}>{t('customer.comboSubtitle')}</p>

      {comboError ? <Notice variant="error">{comboError}</Notice> : null}

      {combosLoading ? <p className={INK_62}>{t('common.loading')}</p> : null}

      {hasCombos ? (
        <div className="mb-6 grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-3.5">
          {combos.map((combo) => {
            const count = qty[combo.id] ?? 0;
            return (
              <div
                key={combo.id}
                className={`flex flex-col overflow-hidden rounded-xl border ${INK_BORDER_14} ${INK_BG_035}`}
              >
                <div className={`flex h-30 items-center justify-center ${INK_BG_06}`}>
                  {combo.image_url ? (
                    <img
                      src={combo.image_url}
                      alt=""
                      loading="lazy"
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <span className={`text-[32px] font-bold ${INK_60}`} aria-hidden="true">
                      {combo.name.charAt(0).toUpperCase()}
                    </span>
                  )}
                </div>
                <div className="flex flex-1 flex-col gap-0.5 px-3.5 pt-3 pb-1.5">
                  <span className="text-[15px] font-bold">{combo.name}</span>
                  {combo.description ? (
                    <span className={`text-xs ${INK_65}`}>{combo.description}</span>
                  ) : null}
                  <span className="mt-1.5 text-sm font-bold text-brand">
                    {formatVND(combo.price)}
                  </span>
                </div>
                <div className="flex items-center justify-center gap-3.5 px-3.5 pt-2.5 pb-3.5">
                  <button
                    type="button"
                    className={STEPPER_BTN_CLASS}
                    onClick={() => onQty(combo.id, -1)}
                    disabled={count === 0}
                    aria-label={t('customer.comboDecrease', { name: combo.name })}
                  >
                    −
                  </button>
                  <span className="min-w-5 text-center text-[15px] font-bold tabular-nums">
                    {count}
                  </span>
                  <button
                    type="button"
                    className={STEPPER_BTN_CLASS}
                    onClick={() => onQty(combo.id, 1)}
                    disabled={count >= maxQty}
                    aria-label={t('customer.comboIncrease', { name: combo.name })}
                  >
                    +
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      ) : null}

      {/* Skip/Continue live IN THE SIDEBAR (parent's onBack/onContinue) - this column is quantities only, no nav buttons. */}
    </>
  );
};

export default ComboStep;
