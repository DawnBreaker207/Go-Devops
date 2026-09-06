import { lazy } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '@/layouts/MainLayout';
import AuthLayout from '@/layouts/AuthLayout';
import ProtectedRoute from './ProtectedRoute';
import { PATHS } from './paths';

const LoginPage = lazy(() => import('@/features/auth/LoginPage'));
const DashboardPage = lazy(() => import('@/features/dashboard/DashboardPage'));
const MoviesPage = lazy(() => import('@/features/movie/MoviesPage'));
const ShowtimesPage = lazy(() => import('@/features/showtime/ShowtimesPage'));
const BookingsPage = lazy(() => import('@/features/booking/BookingsPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

export const router = createBrowserRouter([
  {
    element: <AuthLayout />,
    children: [{ path: PATHS.login, element: <LoginPage /> }],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <MainLayout />,
        children: [
          { path: '/', element: <Navigate to={PATHS.dashboard} replace /> },
          { path: PATHS.dashboard, element: <DashboardPage /> },
          { path: PATHS.movies, element: <MoviesPage /> },
          { path: PATHS.showtimes, element: <ShowtimesPage /> },
          { path: PATHS.bookings, element: <BookingsPage /> },
        ],
      },
    ],
  },
  { path: PATHS.notFound, element: <NotFoundPage /> },
]);
