const DEVICE_ID_KEY = 'cp_device_id';

const newDeviceId = (): string => {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
};

export const getDeviceId = (): string => {
  const existing = localStorage.getItem(DEVICE_ID_KEY);
  if (existing) return existing;

  const generated = newDeviceId();
  localStorage.setItem(DEVICE_ID_KEY, generated);
  return generated;
};
