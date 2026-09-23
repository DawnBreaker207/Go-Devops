import { seatType as seatTypeToken } from '@/theme';
import type { SeatType } from '@/types';

/** Seat-type colors sourced from the theme tokens. */
export const SEAT_TYPE_STYLE: Record<SeatType, { bg: string; fg: string }> = {
  standard: seatTypeToken.standard,
  vip: seatTypeToken.vip,
  couple: seatTypeToken.couple,
  recliner: seatTypeToken.recliner,
};

/** Price form scale: whole VND, no minor units. */
export const MIN_SEAT_PRICE = 1;
export const MAX_SEAT_PRICE = 100_000_000;

/** Backend ceiling on dto.HallRequest (binding min=1, max=50). */
export const MAX_ROWS = 50;
export const MAX_SEATS_PER_ROW = 50;
