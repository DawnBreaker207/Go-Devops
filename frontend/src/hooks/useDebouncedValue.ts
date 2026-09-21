import { useEffect, useState } from 'react';

/** Delays value updates by delayMs after the last change, so search inputs feeding a useQuery key don't fire per keystroke. */
export const useDebouncedValue = <T>(value: T, delayMs = 350): T => {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(timer);
  }, [value, delayMs]);

  return debounced;
};

export default useDebouncedValue;
