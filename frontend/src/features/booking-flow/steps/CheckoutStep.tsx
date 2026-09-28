import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useCountdown } from '../useCountdown';
import { useOrderDetail, usePayOrder, usePaymentProviders } from '../hooks/useOrders';
import { errorMessage, reasonMessage } from '@/utils/error';
import { formatVND } from '@/utils/format';
import Button from '@/components/ui/Button';
import Notice from '@/components/ui/Notice';
import FieldInput from '@/components/ui/FieldInput';
import {
  INK_60,
  INK_62,
  INK_BG_035,
  INK_BG_08,
  INK_BORDER_10,
  INK_BORDER_14,
} from '@/theme/customerTw';
import { useApplyDiscount, useRemoveDiscount } from '../hooks/useDiscount';
import { useComboOrderForBooking } from '../hooks/useCombos';

interface CheckoutStepProps {
  bookingId: string;
  // Sidebar Pay signals this step to pay exactly once per signal.
  paySignal?: number;
}

// Step 3 settles in /payment-result; never trust client-side paid.
export const CheckoutStep = ({ bookingId, paySignal = 0 }: CheckoutStepProps) => {
  const { t } = useTranslation();

  const [actionError, setActionError] = useState<string | null>(null);
  // Discount lives on the order so F5 keeps it; local state holds only the input + display code.
  const [voucherInput, setVoucherInput] = useState('');
  const [appliedCode, setAppliedCode] = useState<string | null>(null);
  const [voucherError, setVoucherError] = useState<string | null>(null);
  const [pickedProvider, setPickedProvider] = useState<string | null>(null);

  const order = useOrderDetail(bookingId);
  const pay = usePayOrder();
  const applyDiscount = useApplyDiscount(bookingId);
  const removeDiscount = useRemoveDiscount(bookingId);
  // Snacks are a separate order the gateway never charges; shown so the counter total is no surprise.
  const { comboOrder } = useComboOrderForBooking(bookingId);

  const isPending = order.data?.status === 'pending';
  const payment = order.data?.payment;
  // A failed attempt keeps pending + seats, so retry is allowed - say so instead of going silent.
  const paymentFailed = payment?.status === 'failed';
  const refunding = payment?.status === 'refund_pending' || payment?.status === 'refunded';

  const secondsLeft = useCountdown(order.data?.expires_at);
  const expired = order.data?.status === 'expired' || (isPending && secondsLeft === 0);
  const settled = order.data?.status === 'refunded' || order.data?.status === 'expired';

  const providers = usePaymentProviders();
  const providerList = providers.data ?? [];
  const activeProvider = pickedProvider ?? providerList.find((p) => p.default)?.name;

  // Confirm lives in /payment-result: this page unloads right after redirect.
  const handlePay = async () => {
    if (!bookingId) return;
    setActionError(null);
    try {
      const result = await pay.mutateAsync({
        bookingId,
        payload: activeProvider ? { provider: activeProvider } : {},
      });
      // Same-tab redirect so the gateway returns the browser to /payment-result.
      window.location.href = result.redirect_url;
    } catch (error) {
      setActionError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  // Ref holds the latest pay closure so the effect depends on paySignal only - no double-fire on re-render.
  const latestPay = useRef(() => {});
  useEffect(() => {
    latestPay.current = () => {
      void handlePay();
    };
  });
  const seenSignal = useRef(paySignal);
  useEffect(() => {
    if (paySignal !== seenSignal.current) {
      seenSignal.current = paySignal;
      latestPay.current();
    }
  }, [paySignal]);

  if (order.error) {
    return <Notice variant="error">{errorMessage(order.error, t('common.somethingWrong'))}</Notice>;
  }
  if (order.isLoading) return <p className={INK_62}>{t('common.loading')}</p>;
  if (!order.data) return null;

  const o = order.data;

  // Read money straight off the order: the gateway charges payable_amount.
  const discount = o.discount_amount;
  const payable = o.payable_amount;
  const hasDiscount = discount > 0;
  // An open checkout locks the discount: POST /pay billed the amount, later price edits go uncaptured.
  const checkoutOpen = payment?.status === 'pending';
  const canEditDiscount = isPending && !expired && !o.paid_at && !checkoutOpen;

  const applyVoucher = async () => {
    const code = voucherInput.trim();
    if (!code) return;
    setVoucherError(null);
    try {
      const result = await applyDiscount.mutateAsync(code);
      setAppliedCode(result.code ?? code.toUpperCase());
      setVoucherInput('');
    } catch (error) {
      // Every refusal shares code 40001, so match on reason, never the English sentence.
      setVoucherError(reasonMessage(error, t, t('customer.voucherInvalid')));
    }
  };

  const clearVoucher = async () => {
    setVoucherError(null);
    try {
      await removeDiscount.mutateAsync();
      setAppliedCode(null);
      setVoucherInput('');
    } catch (error) {
      setVoucherError(reasonMessage(error, t, t('common.somethingWrong')));
    }
  };

  return (
    <div>
      {expired && isPending ? (
        <Notice variant="error" role="alert">
          {t('customer.holdExpired')}
        </Notice>
      ) : null}

      {actionError ? <Notice variant="error">{actionError}</Notice> : null}

      {paymentFailed && isPending ? (
        <Notice variant="error">
          {t('customer.paymentFailed', {
            reason: payment?.status_reason
              ? t(`customer.payReason_${payment.status_reason}`, payment.status_reason)
              : t('customer.payReasonUnknown'),
          })}
        </Notice>
      ) : null}

      {/* Hien status + reason cua order, khong chi cua transaction. */}
      {settled ? (
        <Notice variant="error">
          {t('customer.orderSettled', {
            status: t(`booking.status_${o.status}`),
            reason: o.status_reason
              ? t(`booking.reason_${o.status_reason}`, o.status_reason)
              : t('customer.payReasonUnknown'),
          })}
        </Notice>
      ) : null}

      {refunding ? (
        // Wrong-amount gateway reports flip the backend to refund_pending, then it refunds post-commit.
        <Notice variant="info" role="status">
          {t('customer.refundNotice', {
            status: t(`booking.payment_${payment?.status ?? 'refund_pending'}`),
          })}
        </Notice>
      ) : null}

      <section className="mb-6.5">
        <p className="mt-0 mb-2.5 text-sm font-bold">{t('customer.voucherTitle')}</p>
        {hasDiscount ? (
          <div className="flex items-center justify-between gap-2.5">
            <span className="rounded-md bg-brand px-2 py-0.5 font-mono text-xs font-bold text-on-brand">
              {/* Reload mat chuoi code vi order chi giu so tien, fallback label chung. */}
              {appliedCode ?? t('customer.voucherApplied')}
            </span>
            <span className="text-sm font-bold text-brand tabular-nums">
              −{formatVND(discount)}
            </span>
            <Button
              variant="ghost"
              aria-label={t('customer.voucherRemove')}
              disabled={!canEditDiscount || removeDiscount.isPending}
              onClick={() => void clearVoucher()}
            >
              ✕
            </Button>
          </div>
        ) : (
          <div className="flex gap-2">
            <div className="min-w-0 flex-1">
              <FieldInput
                id="checkout-voucher"
                label={t('customer.voucherPlaceholder')}
                value={voucherInput}
                maxLength={32}
                disabled={!canEditDiscount}
                onChange={(event) => {
                  setVoucherInput(event.target.value);
                  setVoucherError(null);
                }}
                error={voucherError ?? undefined}
              />
            </div>
            <Button
              className="flex-none self-end"
              disabled={!canEditDiscount || !voucherInput.trim() || applyDiscount.isPending}
              onClick={() => void applyVoucher()}
            >
              {t('customer.voucherApply')}
            </Button>
          </div>
        )}
        <div className={`my-3 h-px ${INK_BG_08}`} />
        {/* Chi dong payable la so tien khach tra that. */}
        <div className="flex items-baseline justify-between gap-4 py-1 text-[15px]">
          <span className={INK_60}>{t('customer.voucherSubtotal')}</span>
          <span className="font-semibold tabular-nums">{formatVND(o.total_amount)}</span>
        </div>
        <div className="flex items-baseline justify-between gap-4 py-1 text-[15px]">
          <span className={INK_60}>{t('customer.voucherDiscount')}</span>
          <span className="font-semibold text-brand tabular-nums">
            {hasDiscount ? `−${formatVND(discount)}` : formatVND(0)}
          </span>
        </div>
        <div className="flex items-baseline justify-between gap-4 text-xl font-bold">
          <span>{t('customer.voucherPayable')}</span>
          <span className="tabular-nums">{formatVND(payable)}</span>
        </div>

        {/* Snack nam ngoai total vi combo la order rieng, gateway chi thu tien ve. */}
        {comboOrder && comboOrder.items.length > 0 ? (
          <div className={`mt-4 rounded-xl border p-3 ${INK_BORDER_14} ${INK_BG_035}`}>
            <p className="mt-0 mb-2 text-sm font-bold">{t('customer.comboSummaryTitle')}</p>
            {comboOrder.items.map((item) => (
              <div
                key={item.combo_id}
                className="flex items-baseline justify-between gap-4 py-0.5 text-sm"
              >
                <span className={INK_60}>
                  {item.combo_name} × {item.quantity}
                </span>
                <span className="tabular-nums">{formatVND(item.subtotal)}</span>
              </div>
            ))}
            <div className="mt-2 flex items-baseline justify-between gap-4 text-sm font-bold">
              <span>{t('customer.comboSummaryTotal')}</span>
              <span className="tabular-nums">{formatVND(comboOrder.total)}</span>
            </div>
            <p className={`mt-2 mb-0 text-xs ${INK_60}`}>{t('customer.comboPayAtCounter')}</p>
          </div>
        ) : null}
      </section>

      <section className={`overflow-hidden rounded-2xl border ${INK_BORDER_14} ${INK_BG_035}`}>
        <div className={`flex items-center gap-2.5 border-b px-4 py-3 ${INK_BORDER_10}`}>
          <span
            className="flex h-8 w-8 flex-none items-center justify-center rounded-full bg-brand/10 text-brand"
            aria-hidden="true"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <rect
                x="2.5"
                y="5.5"
                width="19"
                height="13"
                rx="2"
                stroke="currentColor"
                strokeWidth="1.7"
              />
              <path d="M2.5 10h19" stroke="currentColor" strokeWidth="1.7" />
            </svg>
          </span>
          <h2 className="m-0 text-lg font-bold">{t('customer.payMethodTitle')}</h2>
        </div>
        <div className="p-4">
          {providers.isLoading ? (
            <p className={INK_60}>{t('common.loading')}</p>
          ) : (
            <div
              role="radiogroup"
              aria-label={t('customer.payMethodTitle')}
              className="flex flex-col gap-2.5"
            >
              {providerList.map((p) => {
                const checked = activeProvider === p.name;
                return (
                  <label
                    key={p.name}
                    className={`flex cursor-pointer items-center gap-3 rounded-xl border p-3.5 transition-colors duration-fast ease-out ${
                      checked ? 'border-brand bg-brand/8' : `${INK_BORDER_14} ${INK_BG_035}`
                    }`}
                  >
                    <input
                      type="radio"
                      name="cp-pay-provider"
                      value={p.name}
                      checked={checked}
                      onChange={() => setPickedProvider(p.name)}
                      className="h-4 w-4 flex-none accent-brand"
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-bold">{p.display_name}</span>
                      <span className={`block truncate font-mono text-xs ${INK_60}`}>{p.name}</span>
                    </span>
                    {p.default && !pickedProvider ? (
                      <span className={`shrink-0 text-xs ${INK_60}`}>•</span>
                    ) : null}
                  </label>
                );
              })}
            </div>
          )}
        </div>
      </section>
    </div>
  );
};

export default CheckoutStep;
