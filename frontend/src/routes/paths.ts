export const PATHS = {
  login: '/login',
  dashboard: '/dashboard',
  movies: '/movies',
  showtimes: '/showtimes',
  bookings: '/bookings',
  notFound: '*',
} as const;

export type AppPath = (typeof PATHS)[keyof typeof PATHS];
