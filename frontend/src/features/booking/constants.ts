import type { BookingStatus, PaymentStatus } from '@/types';

/** Mau tag cua antd cho tung trang thai don. */
export const BOOKING_STATUS_COLOR: Record<BookingStatus, string> = {
  pending: 'gold',
  confirmed: 'green',
  expired: 'default',
  refunded: 'red',
};

export const PAYMENT_STATUS_COLOR: Record<PaymentStatus, string> = {
  pending: 'gold',
  paid: 'green',
  failed: 'red',
  refund_pending: 'orange',
  refunded: 'purple',
};
