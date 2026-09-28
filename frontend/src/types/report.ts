import type { StaffShowtime } from './staff';

// Mirrors Go admin report DTO.
export interface DailyReport {
  from: string;
  to: string;
  total_revenue: number;
  tickets_sold: number;
  days: DailyAggregate[];
}

export interface DailyAggregate {
  report_date: string;
  total_revenue: number;
  tickets_sold: number;
  seats_sold: number;
  capacity: number;
  occupancy_rate: number;
  breakdown: DailyBreakdown;
  updated_at: string;
}

export interface DailyBreakdown {
  showtimes?: DailyBreakdownShowtime[];
}

export interface DailyBreakdownShowtime {
  showtime_id: string;
  movie: string;
  hall: string;
  start_at: string;
  capacity: number;
  seats_sold: number;
  checked_in: number;
  revenue: number;
}

export interface DailyReportQuery {
  from?: string;
  to?: string;
}

export interface BreakdownQuery {
  from?: string;
  to?: string;
}

export interface BreakdownDay {
  date: string;
  revenue: number;
  tickets: number;
}

export interface BreakdownMovie {
  movie_id: string;
  title: string;
  revenue: number;
  tickets: number;
}

export interface BreakdownHall {
  hall_id: string;
  name: string;
  revenue: number;
  tickets: number;
}

export interface BreakdownProvider {
  provider: string;
  revenue: number;
  count: number;
}

export interface Breakdown {
  from: string;
  to: string;
  total_revenue: number;
  tickets_sold: number;
  days: BreakdownDay[];
  movies: BreakdownMovie[];
  halls: BreakdownHall[];
  providers: BreakdownProvider[];
}

export interface StuckRefundAlert {
  payment_id: string;
  booking_id: string;
  attempts: number;
  amount: number;
  last_error?: string;
}

export interface FailedJobAlert {
  id: string;
  job_name: string;
  error_message?: string;
  started_at: string;
}

export interface GivenUpEmailAlert {
  booking_id: string;
  attempts: number;
  created_at: string;
}

export interface AdminAlerts {
  stuck_refunds: StuckRefundAlert[];
  failed_jobs: FailedJobAlert[];
  given_up_emails: GivenUpEmailAlert[];
}

export interface AdminOverview {
  today: DailyAggregate;
  last_7_days: DailyAggregate[];
  upcoming_showtimes: StaffShowtime[];
  alerts: AdminAlerts;
}
