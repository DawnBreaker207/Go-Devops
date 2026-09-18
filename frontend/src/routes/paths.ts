export const PATHS = {
  // --- Khu khach hang (CustomerLayout) ---
  home: '/',
  film: '/film/:id',
  selectSeat: '/select-seat/:showtimeId',
  checkout: '/checkout/:bookingId',
  orderSuccess: '/checkout/:bookingId/success',
  myTickets: '/my-tickets',
  register: '/register',
  forgotPassword: '/forgot-password',
  resetPassword: '/reset-password',

  // --- Khu van hanh (MainLayout) ---
  login: '/login',
  dashboard: '/dashboard',
  movies: '/movies',
  showtimes: '/showtimes',
  halls: '/halls',
  hallSeats: '/halls/:id/seats',
  bookings: '/bookings',
  users: '/users',
  reports: '/reports',
  notFound: '*',
} as const;

export type AppPath = (typeof PATHS)[keyof typeof PATHS];

/** Dung duong dan so do ghe cua mot phong. Khong noi chuoi '/halls/...' o cho khac. */
export const hallSeatsPath = (id: string): string => `/halls/${id}/seats`;

/* Cac duong dan co tham so cua khu khach. Khong noi chuoi duong dan o cho khac. */
export const filmPath = (id: string): string => `/film/${id}`;
export const selectSeatPath = (showtimeId: string): string => `/select-seat/${showtimeId}`;
export const checkoutPath = (bookingId: string): string => `/checkout/${bookingId}`;
export const orderSuccessPath = (bookingId: string): string => `/checkout/${bookingId}/success`;
