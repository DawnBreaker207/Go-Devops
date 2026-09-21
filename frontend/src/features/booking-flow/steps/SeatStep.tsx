import { useTranslation } from 'react-i18next';
import type { SeatMapSeat } from '@/types';
import type { GridLayout, SeatRow } from '@/features/hall/seatGrid';
import SeatIcon, { type SeatShape } from '../components/SeatIcon';
import SeatButton, { type SeatVisual } from '../components/SeatButton';
import ScreenArch from '../components/ScreenArch';
import Notice from '@/components/ui/Notice';
import { INK_60, INK_65, INK_75, INK_BG_035, INK_BORDER_14 } from '@/theme/customerTw';
import type { SeatMap } from '@/types';

const seatVisual = (seat: SeatMapSeat, selected: boolean): SeatVisual => {
  // Check order matters: gaps and missing showtime_seat_id also report status
  // 'available', so they must win first.
  if (seat.is_gap || !seat.showtime_seat_id) return 'blank';
  if (selected) return 'selected';
  if (seat.status === 'sold') return 'sold';
  if (seat.status === 'held') return 'held';
  return 'available';
};

interface SeatStepProps {
  show: SeatMap;
  rows: SeatRow<SeatMapSeat>[];
  layout: GridLayout;
  selected: Set<string>;
  holdError: string | null;
  isCustomer: boolean;
  /** STABLE (parent useCallback) - what lets `SeatButton` (React.memo) skip re-renders. */
  onToggle: (seatId: string) => void;
}

/** Step 1: seat map. Selection accumulates in the parent; this step only draws grid + legend + notices. */
export const SeatStep = ({
  show,
  rows,
  layout,
  selected,
  holdError,
  isCustomer,
  onToggle,
}: SeatStepProps) => {
  const { t } = useTranslation();
  // Two-group legend: color = status, shape = kind.
  const statusLegend: SeatVisual[] = ['available', 'selected', 'held', 'sold'];
  const typeLegend: Array<{ shape: SeatShape; labelKey: string }> = [
    { shape: 'single', labelKey: 'customer.legendTypeStandard' },
    { shape: 'vip', labelKey: 'customer.legendTypeVip' },
    { shape: 'couple', labelKey: 'customer.legendTypeCouple' },
  ];
  const screenArch = <ScreenArch />;

  return (
    <>
      {!isCustomer ? (
        <Notice variant="info" role="status">
          {t('customer.operatorCannotBook')}
        </Notice>
      ) : null}
      {holdError ? <Notice variant="error">{holdError}</Notice> : null}

      {/* Theme-reactive map card - seats/grid/legend follow the theme. */}
      <div className={`rounded-2xl border p-4 ${INK_BORDER_14} ${INK_BG_035}`}>
        {show.screen_position === 'front' ? screenArch : null}

        <div className="mt-4 overflow-x-auto pt-2 pb-1">
          <div className="inline-flex min-w-full flex-col items-center gap-1.5">
            {rows.map((row) => (
              <div key={row.rowLabel} className="flex items-center gap-2">
                <span className={`w-5.5 text-center text-xs font-bold ${INK_65}`}>
                  {row.rowLabel}
                </span>
                <div
                  className="grid items-stretch gap-1"
                  style={{ gridTemplateColumns: layout.templateColumns }}
                >
                  {row.seats.map((seat) => {
                    const isSelected = Boolean(
                      seat.showtime_seat_id && selected.has(seat.showtime_seat_id)
                    );
                    const visual = seatVisual(seat, isSelected);
                    return (
                      <SeatButton
                        key={seat.id}
                        seatId={seat.showtime_seat_id}
                        label={seat.label}
                        price={seat.price}
                        colNumber={seat.col_number}
                        seatType={seat.seat_type}
                        visual={visual}
                        gridColumn={`${layout.lineOf(seat.col_number)} / span ${layout.spanOf(
                          seat.col_number,
                          seat.col_span
                        )}`}
                        onToggle={onToggle}
                      />
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>

        {show.screen_position === 'back' ? screenArch : null}

        {/* Two-group legend: color = status, shape = kind. */}
        <div
          className={`mt-5 flex flex-col gap-3 border-t pt-4 text-xs sm:flex-row sm:gap-8 ${INK_BORDER_14} ${INK_75}`}
        >
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className={`text-[11px] font-bold tracking-wider uppercase ${INK_60}`}>
              {t('customer.legendStatus')}
            </span>
            {statusLegend.map((key) => (
              <span key={key} className="inline-flex items-center gap-1.5">
                <SeatIcon
                  shape="single"
                  outline={key === 'held'}
                  className={`h-5 w-5 flex-none text-(--cp-seat-${key})`}
                />
                {t(`customer.seat_${key}`)}
              </span>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className={`text-[11px] font-bold tracking-wider uppercase ${INK_60}`}>
              {t('customer.legendType')}
            </span>
            {typeLegend.map((item) => (
              <span key={item.shape} className="inline-flex items-center gap-1.5">
                <SeatIcon
                  shape={item.shape}
                  className="h-5 w-5 flex-none text-(--cp-seat-available)"
                />
                {t(item.labelKey)}
              </span>
            ))}
          </div>
        </div>
      </div>
    </>
  );
};

export default SeatStep;
