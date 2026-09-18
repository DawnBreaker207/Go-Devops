import { lazy } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '@/layouts/MainLayout';
import AuthLayout from '@/layouts/AuthLayout';
import ProtectedRoute from './ProtectedRoute';
import RequireRole from './RequireRole';
import { ROLES_ADMIN, ROLES_OPERATOR } from './navigation';
import { PATHS } from './paths';

const LoginPage = lazy(() => import('@/features/auth/LoginPage'));
const DashboardPage = lazy(() => import('@/features/dashboard/DashboardPage'));
const MoviesPage = lazy(() => import('@/features/movie/MoviesPage'));
const ShowtimesPage = lazy(() => import('@/features/showtime/ShowtimesPage'));
const HallsPage = lazy(() => import('@/features/hall/HallsPage'));
const HallSeatsPage = lazy(() => import('@/features/hall/HallSeatsPage'));
const BookingsPage = lazy(() => import('@/features/booking/BookingsPage'));
const UsersPage = lazy(() => import('@/features/user/UsersPage'));
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
          // Nhom theo dung hang so roles ma navigation.tsx dung cho sider, nen
          // menu va quyen vao route khong the lech nhau.
          {
            element: <RequireRole roles={ROLES_OPERATOR} />,
            children: [
              { path: PATHS.dashboard, element: <DashboardPage /> },
              { path: PATHS.movies, element: <MoviesPage /> },
              { path: PATHS.showtimes, element: <ShowtimesPage /> },
              { path: PATHS.halls, element: <HallsPage /> },
              { path: PATHS.hallSeats, element: <HallSeatsPage /> },
            ],
          },
          {
            element: <RequireRole roles={ROLES_ADMIN} />,
            children: [
              { path: PATHS.bookings, element: <BookingsPage /> },
              { path: PATHS.users, element: <UsersPage /> },
            ],
          },
        ],
      },
    ],
  },
  { path: PATHS.notFound, element: <NotFoundPage /> },
]);
