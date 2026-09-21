import type { StaffShowtime } from './staff';

/** Admin only. Read carefully: (1) `days` holds only closeDay-closed dates (daily_aggregates); absent != zero day (zero-business days ARE closed with zeros; absent means the job hasn't run). (2) Revenue/occupancy use different axes: total_revenue/tickets_sold count `confirmed` orders by paid_at day, seats_sold/capacity count shows starting that day (a ticket bought today for tomorrow lifts today's revenue and tomorrow's occupancy). (3) occupancy_rate is already percent (ROUND(100.0 * ...)); never *100. */
export interface DailyReport {
  /** YYYY-MM-DD in cinema time, not RFC3339. */
  from: string;
  to: string;
  /** int64 whole VND. Whole-range total, not per `days`. */
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
  /** Already percent; see note above. */
  occupancy_rate: number;
  breakdown: DailyBreakdown;
  /** Lan cuoi job chot so ngay nay. RFC3339. */
  updated_at: string;
}

/** `breakdown` is free-form jsonb, not a constrained struct. Backend currently writes exactly one key `showtimes` (COALESCE to `[]`, always present), but nothing in the schema pins that, so the field is optional and every reader must tolerate absence. */
export interface DailyBreakdown {
  showtimes?: DailyBreakdownShowtime[];
}

export interface DailyBreakdownShowtime {
  showtime_id: string;
  /** Movie/hall names snapshotted at close; later renames don't rewrite old reports, by design. */
  movie: string;
  hall: string;
  start_at: string;
  capacity: number;
  seats_sold: number;
  checked_in: number;
  revenue: number;
}

/** Empty both = last 7 days. */
export interface DailyReportQuery {
  /** Defaults to `to` - 6 days. */
  from?: string;
  /** Defaults to today in cinema time. */
  to?: string;
}

/** Paid-money analytics over [from, to]: daily line, top movies/halls, payment split. Same money rule as closeDay, aggregated live for any range. */
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

/** One call for the whole dashboard: today (live, not closeDay), last 7 closed days for trend, today's remaining shows, ops alerts. */
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

/** Seat/check-in shape of one show; the staff board shares it. */
