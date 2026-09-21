import type { ShowtimeStatus } from './showtime';

/** Mirrors StaffShowtimeResponse/Board/BoxOfficeDay/Overview/TicketResponse plus CounterSellRequest. All /staff/* require staff or admin (see router.go). */

/** One show on the staff board. Seat counts only; no money on this screen. */
export interface StaffShowtime {
  id: string;
  movie_title: string;
  hall_name: string;
  start_at: string;
  end_at: string;
  status: ShowtimeStatus;
  capacity: number;
  held: number;
  sold: number;
  available: number;
  checked_in: number;
}

/** One day's board. */
export interface StaffBoard {
  date: string;
  showtimes: StaffShowtime[];
}

/** Counter-only sales for the day; excludes online. */
export interface BoxOfficeDay {
  date: string;
  count: number;
  total: number;
}

/** Board + counter sales + pending-checkin count in one call. */
export interface StaffOverview {
  date: string;
  showtimes: StaffShowtime[];
  counter_sales_count: number;
  counter_sales_total: number;
  awaiting_checkin: number;
}

export type StaffTicketStatus = 'issued' | 'redeemed';

/** One row of GET /staff/showtimes/:id/tickets. */
export interface StaffTicket {
  id: string;
  booking_id: string;
  seat_label: string;
  seat_type: string;
  status: StaffTicketStatus;
  updated_at: string;
}

/** Counter sale: no account, no gateway, confirmed immediately, no email. seat_ids must be showtime_seat_id from GET /shows/:id/seats. */
export interface CounterSellPayload {
  show_id: string;
  seat_ids: string[];
  customer_name?: string;
  customer_phone?: string;
}
