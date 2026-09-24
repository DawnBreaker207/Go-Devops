/** Browser device_id for login/sessions (backend max=255). Generated once, reused in localStorage; marks `is_current` in GET /users/me/sessions. */
const DEVICE_ID_KEY = 'cp_device_id';

// randomUUID needs HTTPS/localhost; fallback for plain http.
const newUuid = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
};

export const getDeviceId = (): string => {
  const existing = localStorage.getItem(DEVICE_ID_KEY);
  if (existing) return existing;

  const generated = newUuid();
  localStorage.setItem(DEVICE_ID_KEY, generated);
  return generated;
};
