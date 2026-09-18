import { seatType as seatTypeToken } from '@/theme';
import type { SeatType } from '@/types';

/**
 * Mau cua tung loai ghe trong trinh sua so do. Lay tu token, khong viet ma mau
 * o day - xem `.claude/rules/design-tokens.md`.
 */
export const SEAT_TYPE_STYLE: Record<SeatType, { bg: string; fg: string }> = {
  standard: seatTypeToken.standard,
  vip: seatTypeToken.vip,
  couple: seatTypeToken.couple,
  recliner: seatTypeToken.recliner,
};

/** Kich thuoc form gia: VND nguyen, khong co don vi phu. */
export const MIN_SEAT_PRICE = 1;
export const MAX_SEAT_PRICE = 100_000_000;

/** Tran cua backend tren dto.HallRequest (binding min=1, max=50). */
export const MAX_ROWS = 50;
export const MAX_SEATS_PER_ROW = 50;
