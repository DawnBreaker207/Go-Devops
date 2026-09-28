import { seatType as seatTypeToken } from '@/theme';
import type { SeatType } from '@/types';

/** Seat-type colors sourced from the theme tokens. */
export const SEAT_TYPE_STYLE: Record<SeatType, { bg: string; fg: string }> = {
  standard: seatTypeToken.standard,
  vip: seatTypeToken.vip,
  couple: seatTypeToken.couple,
  recliner: seatTypeToken.recliner,
};

/** Backend ceiling on dto.HallRequest (binding min=1, max=50). */
export const MAX_ROWS = 50;
export const MAX_SEATS_PER_ROW = 50;

/** Quick-create's starting row width - 1 would trap every row at a single seat. */
export const QUICK_CREATE_SEATS_PER_ROW = 10;
