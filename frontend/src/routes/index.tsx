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
const MoviesPage = lazy(() => import('@/features/movie/MoviesPage'));
const ShowtimesPage = lazy(() => import('@/features/showtime/ShowtimesPage'));
const HallsPage = lazy(() => import('@/features/hall/HallsPage'));
const ConcessionsPage = lazy(() => import('@/features/concession/ConcessionsPage'));
const DiscountsPage = lazy(() => import('@/features/discount/DiscountsPage'));
const BookingsPage = lazy(() => import('@/features/booking/BookingsPage'));
const UsersPage = lazy(() => import('@/features/user/UsersPage'));
const ReportsPage = lazy(() => import('@/features/report/ReportsPage'));
const BoxOfficePage = lazy(() => import('@/features/staff/BoxOfficePage'));
const CustomerLookupPage = lazy(() => import('@/features/staff/CustomerLookupPage'));
const AuditLogPage = lazy(() => import('@/features/audit/AuditLogPage'));
const BatchJobsPage = lazy(() => import('@/features/batch/BatchJobsPage'));
const HomePage = lazy(() => import('@/features/browse/HomePage'));
const FilmPage = lazy(() => import('@/features/browse/FilmPage'));
const FilmsPage = lazy(() => import('@/features/browse/FilmsPage'));
const PricingPage = lazy(() => import('@/features/browse/PricingPage'));
const CinemaInfoPage = lazy(() => import('@/features/browse/CinemaInfoPage'));
const OffersPage = lazy(() => import('@/features/browse/OffersPage'));
const AccountPage = lazy(() => import('@/features/browse/AccountPage'));
// Booking flow is a single route with internal step state; success is a separate page.
// See BookingFlowPage.tsx / BookingSuccessPage.tsx.
const BookingFlowPage = lazy(() => import('@/features/booking-flow/BookingFlowPage'));
const BookingSuccessPage = lazy(() => import('@/features/booking-flow/BookingSuccessPage'));
const PaymentResultPage = lazy(() => import('@/features/booking-flow/PaymentResultPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

export const router = createBrowserRouter([
  // Customer zone. Browsing movies/picking showtimes is public (backend OptionalAuth), no ProtectedRoute wrapper.
  {
    element: <CustomerLayout />,
    children: [
      // `fullBleed` lets the QVisionShow hero reach the viewport edges; it drops <main>'s
      // max-width AND padding, so every section in HomePage carries its own HOME_SECTION container.
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
      { path: PATHS.account, element: <AccountPage /> },
      // Single `/select-seat/:showtimeId` route, internal step state.
      // `handle.forceDark` pins the whole booking flow to dark whatever the customer's switch says:
      // the seat palette (white screen bar, pale-grey free seats) is specified against the dark
      // backdrop, and light mode can only approximate it with substitutions.
      { path: PATHS.selectSeat, element: <BookingFlowPage />, handle: { forceDark: true } },
      // Post-payment landing: separate page outside the flow; clears stale flow store on mount.
      { path: PATHS.bookingSuccess, element: <BookingSuccessPage />, handle: { forceDark: true } },
      // 303 redirect from the gateway: verify, then confirm.
      { path: PATHS.paymentResult, element: <PaymentResultPage />, handle: { forceDark: true } },
      // Legacy /my-tickets route: tickets now live in Account, redirect keeps old bookmarks working.
      { path: '/my-tickets', element: <Navigate to={accountTicketsPath()} replace /> },
      // Customer account screen (Figma 77-626) sits in CustomerLayout to share
      // header/footer. `handle.fullBleed` drops <main> max-width/padding for the split panel.
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
          // Groups reuse the same roles constants as the sider so menu and access never drift apart.
          {
            element: <RequireRole roles={ROLES_OPERATOR} />,
            children: [
              { path: PATHS.dashboard, element: <DashboardPage /> },
              { path: PATHS.profile, element: <ProfilePage /> },
              { path: PATHS.movies, element: <MoviesPage /> },
              { path: PATHS.showtimes, element: <ShowtimesPage /> },
              { path: PATHS.halls, element: <HallsPage /> },
              { path: PATHS.concessions, element: <ConcessionsPage /> },
              { path: PATHS.boxOffice, element: <BoxOfficePage /> },
              { path: PATHS.customerLookup, element: <CustomerLookupPage /> },
            ],
          },
          {
            element: <RequireRole roles={ROLES_ADMIN} />,
            children: [
              { path: PATHS.bookings, element: <BookingsPage /> },
              { path: PATHS.users, element: <UsersPage /> },
              { path: PATHS.discounts, element: <DiscountsPage /> },
              { path: PATHS.reports, element: <ReportsPage /> },
              { path: PATHS.auditLogs, element: <AuditLogPage /> },
              { path: PATHS.batchJobs, element: <BatchJobsPage /> },
            ],
          },
        ],
      },
    ],
  },
  { path: PATHS.notFound, element: <NotFoundPage /> },
]);
