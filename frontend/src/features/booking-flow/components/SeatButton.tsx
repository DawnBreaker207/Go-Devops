import { memo } from 'react';
import SeatIcon, { type SeatShape } from './SeatIcon';
import { formatVND } from '@/utils/format';
import type { SeatType } from '@/types';

export type SeatVisual = 'available' | 'selected' | 'held' | 'sold' | 'blank';

/** Text color = seat STATUS (`--cp-seat-*` flips per mode in index.css). */
const STATUS_COLOR: Record<Exclude<SeatVisual, 'blank'>, string> = {
  available: 'text-(--cp-seat-available)',
  selected: 'text-(--cp-seat-selected)',
  held: 'text-(--cp-seat-held)',
  sold: 'text-(--cp-seat-sold)',
};

/** Seat NUMBER color, paired with the fill above. Every state has its own `*Text` token for a reason:
 *  on `available` (#d1d5db) white measures ~1.3:1 and the number effectively disappears.
 *  `held` is the one outline icon, so its number sits on the page backdrop, not on a filled seat. */
const STATUS_LABEL_COLOR: Record<Exclude<SeatVisual, 'blank'>, string> = {
  available: 'text-(--cp-seat-available-text)',
  selected: 'text-(--cp-seat-selected-text)',
  held: 'text-(--cp-seat-held)',
  sold: 'text-(--cp-seat-sold-text)',
};

/** Shape = seat KIND: vip/couple read instantly, rest are singles. */
const shapeOf = (seatType: SeatType): SeatShape =>
  seatType === 'vip' ? 'vip' : seatType === 'couple' ? 'couple' : 'single';

/** 36px seat button: tub icon base, centered number. `blank` (gap/unsellable) is transparent space. */
const SEAT_BUTTON_BASE =
  'relative h-9 w-9 min-w-9 rounded-md border-0 p-0 cursor-pointer select-none ' +
  'transition-transform duration-fast ease-out active:scale-[var(--motion-scale-press)] ' +
  'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand';

const SEAT_BUTTON_STATE: Record<SeatVisual, string> = {
  // Highlight by seat color ONLY - no border/bg/scale on hover/select.
  available: 'cursor-pointer',
  selected: '',
  sold: 'cursor-not-allowed opacity-90',
  held: 'cursor-not-allowed opacity-90',
  blank: 'bg-transparent cursor-default pointer-events-none',
};

interface SeatButtonProps {
  /** `showtime_seat_id` - `undefined` for unselectable seats (gap/missing id); `visual` is already 'blank' and `onToggle` never fires. */
  seatId: string | undefined;
  label: string;
  price: number;
  colNumber: number;
  seatType: SeatType;
  visual: SeatVisual;
  gridColumn: string;
  /** STABLE (`useCallback` in parent) - required for `React.memo` below to actually skip re-renders, see BookingFlowPage.tsx. */
  onToggle: (seatId: string) => void;
}

/** Memoized seat cell: one click re-renders one seat, never the 100+ grid. Props are primitives only (shallow compare survives parent array remakes). */
const SeatButtonImpl = ({
  seatId,
  label,
  price,
  colNumber,
  seatType,
  visual,
  gridColumn,
  onToggle,
}: SeatButtonProps) => {
  const disabled = visual !== 'available' && visual !== 'selected';

  return (
    <button
      type="button"
      className={`${SEAT_BUTTON_BASE} ${SEAT_BUTTON_STATE[visual]}`}
      style={{ gridColumn }}
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
            className={`absolute inset-0 flex items-center justify-center text-xs font-bold ${STATUS_LABEL_COLOR[visual]}`}
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
