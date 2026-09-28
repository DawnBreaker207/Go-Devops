import type { ShowtimeStatus } from './showtime';

// Mirrors Go staff board DTO.
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

export interface StaffBoard {
  date: string;
  showtimes: StaffShowtime[];
}

export interface BoxOfficeDay {
  date: string;
  count: number;
  total: number;
}

export interface StaffOverview {
  date: string;
  showtimes: StaffShowtime[];
  counter_sales_count: number;
  counter_sales_total: number;
  awaiting_checkin: number;
}

export type StaffTicketStatus = 'issued' | 'redeemed';

export interface StaffTicket {
  id: string;
  booking_id: string;
  seat_label: string;
  seat_type: string;
  status: StaffTicketStatus;
  updated_at: string;
}

export interface CounterSellPayload {
  show_id: string;
  seat_ids: string[];
  customer_name?: string;
  customer_phone?: string;
}
