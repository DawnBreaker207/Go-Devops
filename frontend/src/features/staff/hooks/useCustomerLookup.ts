import { useQuery } from '@tanstack/react-query';
import { staffApi } from '@/api/staff.api';
import type { PageQuery } from '@/types';

export const CUSTOMER_LOOKUP_QUERY_KEY = 'staff-customers';

/** GET /staff/customers - search role=customer accounts (email/name/phone). */
export const useCustomerList = (query: PageQuery) =>
  useQuery({
    queryKey: [CUSTOMER_LOOKUP_QUERY_KEY, query],
    queryFn: () => staffApi.customers(query),
    placeholderData: (previous) => previous,
  });

/** GET /staff/customers/:id - empty id means the drawer is closed; fetch nothing. */
export const useCustomerProfile = (id: string | null) =>
  useQuery({
    queryKey: [CUSTOMER_LOOKUP_QUERY_KEY, 'profile', id],
    queryFn: () => staffApi.customerProfile(id as string),
    enabled: Boolean(id),
  });

/** GET /staff/customers/:id/orders - order history of the customer open in the drawer. */
export const useCustomerOrders = (id: string | null, query: PageQuery) =>
  useQuery({
    queryKey: [CUSTOMER_LOOKUP_QUERY_KEY, 'orders', id, query],
    queryFn: () => staffApi.customerOrders(id as string, query),
    enabled: Boolean(id),
    placeholderData: (previous) => previous,
  });
