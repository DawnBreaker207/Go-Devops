import type { TFunction } from 'i18next';
import { errorMessage } from '@/utils/error';

/** Backend sentence -> i18n key.
 *
 *  Keyed by MESSAGE, not by code, and that is not a shortcut: every refusal here
 *  is 400/40001 (or 409/40900), so the code cannot tell them apart. The sentences
 *  come from the `ErrDiscount*` sentinels in pkg/errors/errors.go; if one is
 *  reworded there, this map silently falls through to the server's own words
 *  rather than showing a wrong reason.
 *
 *  Scope is deliberately this one flow. Whether the whole app gets a
 *  message-keyed table is still an open decision for the owner (the hall screen
 *  has the same problem with three different 40900 causes) — but a customer one
 *  click from paying should not be shown English, so the discount flow does not
 *  wait for it. */
const MESSAGE_KEYS: Record<string, string> = {
  'this discount code is not valid': 'customer.discountInvalid',
  'this discount code is no longer valid': 'customer.discountExpired',
  'this discount code is not active yet': 'customer.discountNotStarted',
  'this discount code has been fully redeemed': 'customer.discountExhausted',
  "the order total is below this code's minimum": 'customer.discountMinOrder',
  'this order already has a discount code': 'customer.discountAlreadySet',
  'this order has no discount code to remove': 'customer.discountNone',
  'a discount can only be applied before payment': 'customer.discountOrderClosed',
};

/** Vietnamese sentence for a known refusal, else whatever the server said. */
export const discountErrorMessage = (error: unknown, t: TFunction, fallback: string): string => {
  const raw = errorMessage(error, fallback);
  const key = MESSAGE_KEYS[raw.trim().toLowerCase()];
  return key ? t(key) : raw;
};
