import { lazy } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '@/layouts/MainLayout';
import AuthLayout from '@/layouts/AuthLayout';
import CustomerLayout from '@/layouts/CustomerLayout';
import ProtectedRoute from './ProtectedRoute';
import RequireRole from './RequireRole';
import { ROLES_ADMIN, ROLES_OPERATOR } from './navigation';
import { PATHS, accountTicketsPath } from './paths';

const LoginPage = lazy(() => import('@/features/auth/LoginPage'));
const CustomerLoginPage = lazy(() => import('@/features/auth/CustomerLoginPage'));
const RegisterPage = lazy(() => import('@/features/auth/RegisterPage'));
const ForgotPasswordPage = lazy(() => import('@/features/auth/ForgotPasswordPage'));
const ResetPasswordPage = lazy(() => import('@/features/auth/ResetPasswordPage'));
const DashboardPage = lazy(() => import('@/features/dashboard/DashboardPage'));
const ProfilePage = lazy(() => import('@/features/profile/ProfilePage'));
const BookingsPage = lazy(() => import('@/features/booking/BookingsPage'));
const UsersPage = lazy(() => import('@/features/user/UsersPage'));
const CatalogPage = lazy(() => import('@/features/catalog/CatalogPage'));
const CounterPage = lazy(() => import('@/features/counter/CounterPage'));
const PromotionPage = lazy(() => import('@/features/promotion/PromotionPage'));
const PricingAdminPage = lazy(() => import('@/features/pricing/PricingPage'));
const MonitoringPage = lazy(() => import('@/features/monitoring/MonitoringPage'));
const HomePage = lazy(() => import('@/features/browse/HomePage'));
const FilmPage = lazy(() => import('@/features/browse/FilmPage'));
const FilmsPage = lazy(() => import('@/features/browse/FilmsPage'));
const PricingPage = lazy(() => import('@/features/browse/PricingPage'));
const CinemaInfoPage = lazy(() => import('@/features/browse/CinemaInfoPage'));
const OffersPage = lazy(() => import('@/features/browse/OffersPage'));
const OfferDetailPage = lazy(() => import('@/features/browse/OfferDetailPage'));
const AccountPage = lazy(() => import('@/features/account/AccountPage'));
const BookingFlowPage = lazy(() => import('@/features/booking-flow/BookingFlowPage'));
const BookingSuccessPage = lazy(() => import('@/features/booking-flow/BookingSuccessPage'));
const PaymentResultPage = lazy(() => import('@/features/booking-flow/PaymentResultPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

export const router = createBrowserRouter([
  {
    element: <CustomerLayout />,
    children: [
      {
        path: PATHS.home,
        element: <HomePage />,
        handle: { fullBleed: true, overlayHeader: true },
      },
      { path: PATHS.film, element: <FilmPage /> },
      { path: PATHS.films, element: <FilmsPage /> },
      { path: PATHS.pricing, element: <PricingPage /> },
      { path: PATHS.cinemaInfo, element: <CinemaInfoPage /> },
      { path: PATHS.offers, element: <OffersPage /> },
      { path: PATHS.offerDetail, element: <OfferDetailPage /> },
      { path: PATHS.account, element: <AccountPage /> },
      { path: PATHS.selectSeat, element: <BookingFlowPage />, handle: { forceDark: true } },
      { path: PATHS.bookingSuccess, element: <BookingSuccessPage />, handle: { forceDark: true } },
      { path: PATHS.paymentResult, element: <PaymentResultPage />, handle: { forceDark: true } },
      { path: '/my-tickets', element: <Navigate to={accountTicketsPath()} replace /> },
      {
        path: PATHS.customerLogin,
        element: <CustomerLoginPage />,
        handle: { fullBleed: true },
      },
      { path: PATHS.register, element: <RegisterPage />, handle: { fullBleed: true } },
      {
        path: PATHS.forgotPassword,
        element: <ForgotPasswordPage />,
        handle: { fullBleed: true },
      },
      {
        path: PATHS.resetPassword,
        element: <ResetPasswordPage />,
        handle: { fullBleed: true },
      },
    ],
  },
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
          {
            element: <RequireRole roles={ROLES_OPERATOR} />,
            children: [
              { path: PATHS.dashboard, element: <DashboardPage /> },
              { path: PATHS.profile, element: <ProfilePage /> },
              { path: PATHS.catalog, element: <CatalogPage /> },
              { path: PATHS.counter, element: <CounterPage /> },
            ],
          },
          {
            element: <RequireRole roles={ROLES_ADMIN} />,
            children: [
              { path: PATHS.bookings, element: <BookingsPage /> },
              { path: PATHS.users, element: <UsersPage /> },
              { path: PATHS.promotions, element: <PromotionPage /> },
              { path: PATHS.pricingAdmin, element: <PricingAdminPage /> },
              { path: PATHS.monitoring, element: <MonitoringPage /> },
            ],
          },
        ],
      },
    ],
  },
  { path: PATHS.notFound, element: <NotFoundPage /> },
]);
