import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useCountdown } from '../useCountdown';
import { useOrderDetail, usePayOrder, usePaymentProviders } from '../hooks/useOrders';
import { errorMessage } from '@/utils/error';
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
import { MOCK_VOUCHERS, calcMockVoucher } from '../mockVoucher';

interface CheckoutStepProps {
  bookingId: string;
  /** Fires on every sidebar "Pay" press - this step pays exactly once per signal. */
  paySignal?: number;
}

/** Step 3: payment. Picks a gateway + pays; results land on `/payment-result` (same-tab redirect, server-verified, confirmed there). Never trust client-side "paid". */
export const CheckoutStep = ({ bookingId, paySignal = 0 }: CheckoutStepProps) => {
  const { t } = useTranslation();

  const [actionError, setActionError] = useState<string | null>(null);
  // Display-only MOCK voucher: no BE call, never changes real tender.
  const [voucherInput, setVoucherInput] = useState('');
  const [appliedCode, setAppliedCode] = useState<string | null>(null);
  const [voucherInvalid, setVoucherInvalid] = useState(false);
  // Gateway picked by the user (backend default when empty).
  const [pickedProvider, setPickedProvider] = useState<string | null>(null);

  const order = useOrderDetail(bookingId);
  const pay = usePayOrder();

  const isPending = order.data?.status === 'pending';
  // Payment state on entry (e.g. a failed attempt returning here) - no polling
  // anymore: results land on /payment-result.
  const payment = order.data?.payment;
  /** A failed attempt keeps the order pending with seats held, so retry is allowed - but say so instead of a silent clock. */
  const paymentFailed = payment?.status === 'failed';
  const refunding = payment?.status === 'refund_pending' || payment?.status === 'refunded';

  const secondsLeft = useCountdown(order.data?.expires_at);
  const expired = order.data?.status === 'expired' || (isPending && secondsLeft === 0);
  /** The order is terminal, nothing left to do. */
  const settled = order.data?.status === 'refunded' || order.data?.status === 'expired';

  const providers = usePaymentProviders();
  const providerList = providers.data ?? [];
  const activeProvider = pickedProvider ?? providerList.find((p) => p.default)?.name;

  // Confirming happens on /payment-result (this page unloads right after redirect - nowhere left to confirm here).
  const handlePay = async () => {
    if (!bookingId) return;
    setActionError(null);
    try {
      // User-picked gateway (backend default when empty).
      const result = await pay.mutateAsync({
        bookingId,
        payload: activeProvider ? { provider: activeProvider } : {},
      });
      // SAME-TAB gateway redirect - the gateway returns the browser to /payment-result.
      window.location.href = result.redirect_url;
    } catch (error) {
      setActionError(errorMessage(error, t('common.somethingWrong')));
    }
  };

  // No Cancel button by request - pending orders end by TTL or payment only.

  // Sidebar "Pay" calls pay here (same-tab gateway redirect). Ref holds the
  // latest closure so the effect depends on paySignal only - no double-fire
  // on re-render.
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

  const appliedVoucher = appliedCode ? calcMockVoucher(o.total_amount, appliedCode) : null;
  const payable = o.total_amount - (appliedVoucher?.discount ?? 0);

  const applyVoucher = () => {
    const found = calcMockVoucher(o.total_amount, voucherInput);
    setVoucherInvalid(!found);
    if (found) setAppliedCode(found.code);
  };

  return (
    <div>
      {/* Countdown lives in the sidebar (all steps) - here only the already-expired-but-pending alert. */}
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

      {/*
        Settled orders (refunded/expired): state the ORDER's own status AND
        reason, not just the transaction's - e.g. `amount_mismatch` means the
        gateway reported a different amount, auto-refunded with seats released.
      */}
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
        // Refund path: gateway reported a different amount. The backend flags
        // refund_pending then calls the provider post-commit - nothing for the customer to do, just say so.
        <Notice variant="info" role="status">
          {t('customer.refundNotice', {
            status: t(`booking.payment_${payment?.status ?? 'refund_pending'}`),
          })}
        </Notice>
      ) : null}

      <section className="mb-6.5">
        {/* Display-only mock voucher - TRANSPARENT, no box. */}
        <p className="mt-0 mb-2.5 text-sm font-bold">{t('customer.voucherTitle')}</p>
        {appliedVoucher ? (
          <div className="flex items-center justify-between gap-2.5">
            <span className="rounded-md bg-brand px-2 py-0.5 font-mono text-xs font-bold text-on-brand">
              {appliedVoucher.code}
            </span>
            <span className="text-sm font-bold text-brand tabular-nums">
              −{formatVND(appliedVoucher.discount)}
            </span>
            <Button
              variant="ghost"
              aria-label={t('customer.voucherRemove')}
              onClick={() => {
                setAppliedCode(null);
                setVoucherInput('');
                setVoucherInvalid(false);
              }}
            >
              ✕
            </Button>
          </div>
        ) : (
          <>
            <div className="flex gap-2">
              <div className="min-w-0 flex-1">
                <FieldInput
                  id="checkout-voucher"
                  label={t('customer.voucherPlaceholder')}
                  value={voucherInput}
                  maxLength={32}
                  onChange={(event) => {
                    setVoucherInput(event.target.value);
                    setVoucherInvalid(false);
                  }}
                  error={voucherInvalid ? t('customer.voucherInvalid') : undefined}
                />
              </div>
              <Button className="flex-none self-end" onClick={applyVoucher}>
                {t('customer.voucherApply')}
              </Button>
            </div>
            <div className="mt-2.5 flex flex-wrap gap-2">
              {MOCK_VOUCHERS.map((v) => (
                <button
                  key={v.code}
                  type="button"
                  onClick={() => {
                    setVoucherInput(v.code);
                    setVoucherInvalid(false);
                  }}
                  className="cursor-pointer rounded-md border border-dashed border-brand/50 bg-transparent px-2 py-1 font-mono text-xs font-bold text-brand"
                >
                  {v.code} · {v.hint}
                </button>
              ))}
            </div>
          </>
        )}
        <p className={`mt-2.5 mb-0 text-xs ${INK_60}`}>{t('customer.voucherDemoNote')}</p>
        <div className={`my-3 h-px ${INK_BG_08}`} />
        {/* Totals always run 3 lines: total / discount / payable. */}
        <div className="flex items-baseline justify-between gap-4 py-1 text-[15px]">
          <span className={INK_60}>{t('customer.voucherDiscount')}</span>
          <span className="font-semibold text-brand tabular-nums">
            {(appliedVoucher?.discount ?? 0) > 0
              ? `−${formatVND(appliedVoucher?.discount ?? 0)}`
              : formatVND(0)}
          </span>
        </div>
        <div className="flex items-baseline justify-between gap-4 py-1 text-[15px]">
          <span className={INK_60}>{t('customer.voucherDiscount')}</span>
          <span className="font-semibold text-brand tabular-nums">
            −{formatVND(appliedVoucher?.discount ?? 0)}
          </span>
        </div>
        <div className="flex items-baseline justify-between gap-4 text-xl font-bold">
          <span>{t('customer.voucherPayable')}</span>
          <span className="tabular-nums">{formatVND(payable)}</span>
        </div>
      </section>

      {/* Payment method: card section (icon header + radio list). Only this block is a card - voucher/totals stay bare. Pick here, sidebar pays with it. */}
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
