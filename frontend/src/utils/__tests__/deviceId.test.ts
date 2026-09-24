import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getDeviceId } from '@/utils/deviceId';

const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe('getDeviceId', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('returns the stored id without generating a new one', () => {
    localStorage.setItem('cp_device_id', 'stored-id');
    expect(getDeviceId()).toBe('stored-id');
  });

  it('uses native crypto.randomUUID when available and persists it', () => {
    const id = getDeviceId();
    expect(id).toMatch(UUID_V4);
    expect(localStorage.getItem('cp_device_id')).toBe(id);
  });

  it('falls back to Math.random UUID when randomUUID is missing (insecure context)', () => {
    vi.stubGlobal('crypto', {});
    const id = getDeviceId();
    expect(id).toMatch(UUID_V4);
    expect(localStorage.getItem('cp_device_id')).toBe(id);
  });

  it('falls back when crypto is entirely missing', () => {
    vi.stubGlobal('crypto', undefined);
    expect(getDeviceId()).toMatch(UUID_V4);
  });
});
