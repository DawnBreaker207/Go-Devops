/** Browser device_id for login/sessions (backend max=255). Generated once, reused in localStorage; marks `is_current` in GET /users/me/sessions. */
const DEVICE_ID_KEY = 'cp_device_id';

export const getDeviceId = (): string => {
  const existing = localStorage.getItem(DEVICE_ID_KEY);
  if (existing) return existing;

  const generated = crypto.randomUUID();
  localStorage.setItem(DEVICE_ID_KEY, generated);
  return generated;
};
