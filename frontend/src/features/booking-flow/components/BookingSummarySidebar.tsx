import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import Button from '@/components/ui/Button';
import { formatDateTime, formatVND } from '@/utils/format';
import { formatCountdown } from '../useCountdown';
import {
  INK_60,
  INK_65,
  INK_BG_035,
  INK_BG_08,
  INK_BORDER_10,
  INK_BORDER_14,
} from '@/theme/customerTw';
import type { BookingStep } from './BookingStepper';

export interface BookingSummarySidebarProps {
  posterUrl?: string;
  movieTitle: string;
  genre?: string;
  hallName: string;
  startAt: string;
  seatLabels: string[];
  total: number;
  step: BookingStep;
  onBack?: () => void;
  /** Omit and the footer is just Back. */
  onContinue?: () => void;
  continueDisabled?: boolean;
  continueLoading?: boolean;
  /** Seconds left before held seats release - present once held. Omit to hide. */
  countdownSeconds?: number;
  /** Continue-button label override. */
  continueLabel?: string;
  /** Extra footer-row slot (currently unused, reserved). */
  extraFooter?: ReactNode;
}

/** Order summary card, sticky right on desktop. Layout: poster header -> hall+showtime -> seats -> total+countdown -> footer. */
export const BookingSummarySidebar = ({
  posterUrl,
  movieTitle,
  genre,
  hallName,
  startAt,
  seatLabels,
  total,
  step,
  onBack,
  onContinue,
  continueDisabled,
  continueLoading,
  countdownSeconds,
  extraFooter,
  continueLabel: continueLabelProp,
}: BookingSummarySidebarProps) => {
  const { t } = useTranslation();
  let continueLabel = continueLabelProp ?? t('customer.proceed');
  if (continueLabelProp === undefined) {
    if (step === 3) continueLabel = t('customer.payNowCta');
    else if (step === 2) continueLabel = t('customer.comboContinue');
    else if (continueLoading) continueLabel = t('common.loading');
  }

  return (
    // Theme-reactive card (INK vars), no fixed-dark background.
    <aside
      className={`w-full overflow-hidden rounded-2xl border shadow-xl lg:sticky lg:top-24 lg:w-95 ${INK_BORDER_14} ${INK_BG_035}`}
    >
      <div
        className="relative flex items-end gap-3 bg-cover bg-center p-4"
        style={posterUrl ? { backgroundImage: `url(${posterUrl})` } : undefined}
      >
        <div className="absolute inset-0 bg-black/55 backdrop-blur-sm" aria-hidden="true" />
        <div
          className="absolute inset-0 bg-linear-to-t from-black/90 via-black/40 to-transparent"
          aria-hidden="true"
        />
        {posterUrl ? (
          <img
            src={posterUrl}
            alt=""
            className="relative z-10 h-20 w-14 flex-none rounded-lg border border-white/30 object-cover shadow-lg"
          />
        ) : null}
        <div className="relative z-10 min-w-0">
          <p className="truncate text-base font-bold text-white">{movieTitle}</p>
          {genre ? <p className="truncate text-xs text-white/70">{genre}</p> : null}
        </div>
      </div>

      <div className="p-4">
        <div
          className={`divide-y divide-[rgb(var(--cp-ink-rgb))]/10 border-b pb-3 ${INK_BORDER_10}`}
        >
          <div className="flex items-center justify-between py-2 text-sm">
            <span className={INK_65}>{t('customer.hallLabel')}</span>
            <span className="font-medium">{hallName}</span>
          </div>
          <div className="flex items-center justify-between py-2 text-sm">
            <span className={INK_65}>{t('customer.schedule')}</span>
            <span className="text-right">
              <span className="block font-bold text-brand">
                {new Date(startAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </span>
              <span className={`block text-xs ${INK_60}`}>{formatDateTime(startAt)}</span>
            </span>
          </div>
        </div>

        <div className="mt-3 rounded-xl border border-brand/40 bg-brand/8 p-3">
          <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
            <span aria-hidden="true">🎟️</span>
            {t('customer.seatsSummaryTitle')}
            {seatLabels.length > 0 ? (
              <span className="ml-auto rounded-full bg-brand px-2 py-0.5 text-[11px] font-bold text-on-brand">
                {seatLabels.length}
              </span>
            ) : null}
          </div>
          {seatLabels.length > 0 ? (
            <div className="flex flex-wrap justify-end gap-1.5">
              {seatLabels.map((label) => (
                <span
                  key={label}
                  className="rounded-md bg-brand-soft px-2 py-0.5 text-xs font-semibold text-brand-active"
                >
                  {label}
                </span>
              ))}
            </div>
          ) : (
            <p className={`text-right text-xs italic ${INK_60}`}>
              {t('customer.seatsSummaryPlaceholder')}
            </p>
          )}
        </div>

        <div className="mt-3 flex items-center justify-between">
          <span className={`text-sm ${INK_65}`}>{t('customer.total')}</span>
          <span className="text-xl font-bold tabular-nums">{formatVND(total)}</span>
        </div>

        {countdownSeconds !== undefined ? (
          <div className="mt-2 flex items-center justify-center gap-2.5 rounded-lg border border-danger/40 bg-danger/10 py-2.5 text-sm">
            <span className="text-base" aria-hidden="true">
              ⏱
            </span>
            <span
              role="timer"
              className="font-mono text-xl font-bold tracking-wider text-danger tabular-nums"
            >
              {formatCountdown(countdownSeconds)}
            </span>
          </div>
        ) : null}
      </div>

      {/* 3-col footer, always rendered so layout never jumps when a button is missing. */}
      {onBack || onContinue ? (
        <div className={`grid grid-cols-3 gap-2 border-t p-3 ${INK_BORDER_10} ${INK_BG_08}`}>
          <div>
            {onBack ? (
              <Button variant="ghost" block onClick={onBack}>
                ← {t('customer.back')}
              </Button>
            ) : null}
          </div>
          <div className="col-span-2">
            {onContinue ? (
              <Button
                block
                variant="primary"
                // Final step ("Pay now") goes green to mark the flow-ENDING action - one-off color, no shared "success" token.
                className={step === 3 ? 'bg-[#16a34a]! hover-fine:bg-[#15803d]!' : ''}
                disabled={continueDisabled}
                onClick={onContinue}
              >
                {continueLabel} →
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}

      {extraFooter}
    </aside>
  );
};

export default BookingSummarySidebar;
