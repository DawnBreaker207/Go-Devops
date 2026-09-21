export const PATHS = {
  // --- Customer zone (CustomerLayout) ---
  home: '/',
  film: '/film/:id',
  // Booking flow is a single route with internal step state; success is a separate page outside it.
  selectSeat: '/select-seat/:showtimeId',
  checkout: '/checkout/:bookingId',
  orderSuccess: '/checkout/:bookingId/success',
  myTickets: '/my-tickets',
  bookingSuccess: '/booking-success/:bookingId',
  // 303 redirect from the gateway (with ?booking_id&status&payment_status): verify the order, then confirm.
  paymentResult: '/payment-result',
  // Named `films` because `movies` is already the admin page (one shared PATHS, no name clash).
  films: '/films',
  pricing: '/pricing',
  cinemaInfo: '/cinema',
  offers: '/offers',
  account: '/account',
  customerLogin: '/customer-login',
  register: '/register',
  forgotPassword: '/forgot-password',
  resetPassword: '/reset-password',

  // --- Operations zone (MainLayout) ---
  login: '/login',
  dashboard: '/dashboard',
  profile: '/profile',
  movies: '/movies',
  showtimes: '/showtimes',
  halls: '/halls',
  hallSeats: '/halls/:id/seats',
  bookings: '/bookings',
  users: '/users',
  reports: '/reports',
  boxOffice: '/box-office',
  customerLookup: '/customer-lookup',
  auditLogs: '/audit-logs',
  batchJobs: '/batch-jobs',
  notFound: '*',
} as const;

export type AppPath = (typeof PATHS)[keyof typeof PATHS];

export const hallSeatsPath = (id: string): string => `/halls/${id}/seats`;
/* Parameterized customer-zone paths. Never concatenate path strings elsewhere. */
export const filmPath = (id: string): string => `/film/${id}`;
export const selectSeatPath = (showtimeId: string): string => `/select-seat/${showtimeId}`;
export const checkoutPath = (bookingId: string): string => `/checkout/${bookingId}`;
export const orderSuccessPath = (bookingId: string): string => `/checkout/${bookingId}/success`;
export const bookingSuccessPath = (bookingId: string): string => `/booking-success/${bookingId}`;
export const ACCOUNT_TICKETS_TAB = 'tickets';
/** "My tickets" lives inside Account: opens /account with the tickets tab preselected. */
export const accountTicketsPath = (): string => `/account?tab=${ACCOUNT_TICKETS_TAB}`;
