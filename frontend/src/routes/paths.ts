export const PATHS = {
  login: '/login',
  dashboard: '/dashboard',
  movies: '/movies',
  showtimes: '/showtimes',
  halls: '/halls',
  hallSeats: '/halls/:id/seats',
  bookings: '/bookings',
  users: '/users',
  notFound: '*',
} as const;

export type AppPath = (typeof PATHS)[keyof typeof PATHS];

/** Dung duong dan so do ghe cua mot phong. Khong noi chuoi '/halls/...' o cho khac. */
export const hallSeatsPath = (id: string): string => `/halls/${id}/seats`;
