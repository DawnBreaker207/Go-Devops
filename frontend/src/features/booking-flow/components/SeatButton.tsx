import { memo } from 'react';
import SeatIcon, { type SeatShape } from './SeatIcon';
import { formatVND } from '@/utils/format';
import type { SeatType } from '@/types';

export type SeatVisual = 'available' | 'selected' | 'held' | 'sold' | 'blank';

const STATUS_COLOR: Record<Exclude<SeatVisual, 'blank'>, string> = {
  available: 'text-(--cp-seat-available)',
  selected: 'text-(--cp-seat-selected)',
  held: 'text-(--cp-seat-held)',
  sold: 'text-(--cp-seat-sold)',
};

// Each state has its own text token: plain white on available grey is unreadable, and held (outline icon) sits on the page backdrop instead of a fill.
const STATUS_LABEL_COLOR: Record<Exclude<SeatVisual, 'blank'>, string> = {
  available: 'text-(--cp-seat-available-text)',
  selected: 'text-(--cp-seat-selected-text)',
  held: 'text-(--cp-seat-held)',
  sold: 'text-(--cp-seat-sold-text)',
};

const shapeOf = (seatType: SeatType): SeatShape =>
  seatType === 'vip' ? 'vip' : seatType === 'couple' ? 'couple' : 'single';

// Width set inline from wide: a fixed w-9 left couple seats hanging off their grid span.
const SEAT_BUTTON_BASE =
  'relative h-9 min-w-9 rounded-md border-0 bg-transparent p-0 appearance-none cursor-pointer select-none ' +
  'transition-transform duration-fast ease-out active:scale-[var(--motion-scale-press)] ' +
  'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand';

const SINGLE_SEAT_WIDTH = '2.25rem';

const SEAT_BUTTON_STATE: Record<SeatVisual, string> = {
  // Highlight by seat color only - no border/bg/scale.
  available: 'cursor-pointer',
  selected: '',
  sold: 'cursor-not-allowed opacity-90',
  held: 'cursor-not-allowed opacity-90',
  blank: 'bg-transparent cursor-default pointer-events-none',
};

interface SeatButtonProps {
  /** `showtime_seat_id` - `undefined` for unpickable seats (gap or missing id); `visual` is already 'blank' and `onToggle` never fires. */
  seatId: string | undefined;
  label: string;
  price: number;
  colNumber: number;
  seatType: SeatType;
  visual: SeatVisual;
  gridColumn: string;
  /** True when spanning multiple tracks: fill the whole span instead of keeping single-seat width. */
  wide: boolean;
  /** STABLE (`useCallback` in parent) - required for `React.memo` below to actually skip re-renders, see BookingFlowPage.tsx. */
  onToggle: (seatId: string) => void;
}

const SeatButtonImpl = ({
  seatId,
  label,
  price,
  colNumber,
  seatType,
  visual,
  gridColumn,
  wide,
  onToggle,
}: SeatButtonProps) => {
  const disabled = visual !== 'available' && visual !== 'selected';

  return (
    <button
      type="button"
      className={`${SEAT_BUTTON_BASE} ${SEAT_BUTTON_STATE[visual]}`}
      style={{ gridColumn, width: wide ? '100%' : SINGLE_SEAT_WIDTH }}
      aria-label={label}
      aria-pressed={visual === 'selected'}
      disabled={disabled}
      title={`${label} · ${formatVND(price)}`}
      onClick={() => {
        if (seatId) onToggle(seatId);
      }}
    >
      {visual === 'blank' ? null : (
        <>
          <SeatIcon
            shape={shapeOf(seatType)}
            outline={visual === 'held'}
            className={`absolute inset-0 h-full w-full ${STATUS_COLOR[visual]}`}
          />
          <span
            className={`absolute inset-0 flex items-start justify-center pt-0.5 text-xs font-bold ${STATUS_LABEL_COLOR[visual]}`}
          >
            {colNumber}
          </span>
        </>
      )}
    </button>
  );
};

export const SeatButton = memo(SeatButtonImpl);

export default SeatButton;
