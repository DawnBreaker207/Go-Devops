import { lazy } from 'react';
import { createBrowserRouter } from 'react-router-dom';
import MainLayout from '@/layouts/MainLayout';
import AuthLayout from '@/layouts/AuthLayout';
import CustomerLayout from '@/layouts/CustomerLayout';
import ProtectedRoute from './ProtectedRoute';
import RequireRole from './RequireRole';
import { ROLES_ADMIN, ROLES_OPERATOR } from './navigation';
import { PATHS } from './paths';

const LoginPage = lazy(() => import('@/features/auth/LoginPage'));
const RegisterPage = lazy(() => import('@/features/auth/RegisterPage'));
const ForgotPasswordPage = lazy(() => import('@/features/auth/ForgotPasswordPage'));
const ResetPasswordPage = lazy(() => import('@/features/auth/ResetPasswordPage'));
const DashboardPage = lazy(() => import('@/features/dashboard/DashboardPage'));
const MoviesPage = lazy(() => import('@/features/movie/MoviesPage'));
const ShowtimesPage = lazy(() => import('@/features/showtime/ShowtimesPage'));
const HallsPage = lazy(() => import('@/features/hall/HallsPage'));
const HallSeatsPage = lazy(() => import('@/features/hall/HallSeatsPage'));
const BookingsPage = lazy(() => import('@/features/booking/BookingsPage'));
const UsersPage = lazy(() => import('@/features/user/UsersPage'));
const ReportsPage = lazy(() => import('@/features/report/ReportsPage'));
const HomePage = lazy(() => import('@/features/browse/HomePage'));
const FilmPage = lazy(() => import('@/features/browse/FilmPage'));
const SelectSeatPage = lazy(() => import('@/features/booking-flow/SelectSeatPage'));
const CheckoutPage = lazy(() => import('@/features/booking-flow/CheckoutPage'));
const OrderSuccessPage = lazy(() => import('@/features/booking-flow/OrderSuccessPage'));
const MyTicketsPage = lazy(() => import('@/features/booking-flow/MyTicketsPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

export const router = createBrowserRouter([
  // Khu khach hang. Duyet phim va chon suat la CONG KHAI (nhom `public` +
  // OptionalAuth ben backend), nen khong boc trong ProtectedRoute.
  {
    element: <CustomerLayout />,
    children: [
      { path: PATHS.home, element: <HomePage /> },
      { path: PATHS.film, element: <FilmPage /> },
      { path: PATHS.selectSeat, element: <SelectSeatPage /> },
      // Ba man duoi deu goi /orders/* (RequireRoles(customer) ben backend).
      // Khong boc them RequireRole o day: khu khach khong dung man 403 cua khu
      // van hanh, va moi man da tu xu ly loi cua no.
      { path: PATHS.checkout, element: <CheckoutPage /> },
      { path: PATHS.orderSuccess, element: <OrderSuccessPage /> },
      { path: PATHS.myTickets, element: <MyTicketsPage /> },
    ],
  },
  {
    element: <AuthLayout />,
    children: [{ path: PATHS.login, element: <LoginPage /> }],
  },
  // Man tai khoan cua KHACH mang vo chia doi rieng (Figma 77-626), khong dung
  // AuthLayout cua khu van hanh, nen dung o muc goc chu khong boc layout nao.
  { path: PATHS.register, element: <RegisterPage /> },
  { path: PATHS.forgotPassword, element: <ForgotPasswordPage /> },
  { path: PATHS.resetPassword, element: <ResetPasswordPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <MainLayout />,
        children: [
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
              { path: PATHS.reports, element: <ReportsPage /> },
            ],
          },
        ],
      },
    ],
  },
  { path: PATHS.notFound, element: <NotFoundPage /> },
]);
