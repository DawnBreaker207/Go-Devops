import { describe, expect, it, vi } from 'vitest';
import { safeMessage } from '@/utils/error';

const api = (message: string) => ({ code: 50000, message });

describe('safeMessage', () => {
  it('passes through ordinary backend messages', () => {
    expect(safeMessage(api('Seat is already held'), 'Fallback')).toBe('Seat is already held');
  });

  it('hides SQL internals behind the fallback', () => {
    const err = api(
      'upsert daily aggregate: ERROR: there is no unique constraint (SQLSTATE 42P10)'
    );
    expect(safeMessage(err, 'Fallback')).toBe('Fallback');
  });

  it('falls back for non-API errors', () => {
    expect(safeMessage(new Error('boom'), 'Fallback')).toBe('Fallback');
    expect(safeMessage(undefined, 'Fallback')).toBe('Fallback');
  });

  it('logs the raw text for devs', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    safeMessage(api('pq: something broke'), 'Fallback');
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });
});
