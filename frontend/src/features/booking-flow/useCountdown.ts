import { useEffect, useState } from 'react';

const remaining = (deadline: string | undefined): number => {
  if (!deadline) return 0;
  // Never slice or compare strings: one instant arrives with two offsets
  // (+07:00 here, Z there). Date.parse handles both.
  const ms = Date.parse(deadline) - Date.now();
  return ms > 0 ? Math.floor(ms / 1000) : 0;
};

/** Seconds left to a deadline (0 past it, never negative). Recomputed every render, so deadline changes apply instantly with no setState-in-effect. */
export const useCountdown = (deadline: string | undefined): number => {
  const [, setTick] = useState(0);

  useEffect(() => {
    if (!deadline) return;
    const id = setInterval(() => setTick((n) => n + 1), 1000);
    return () => clearInterval(id);
  }, [deadline]);

  return remaining(deadline);
};

/** 125 -> "02:05". */
export const formatCountdown = (seconds: number): string => {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
};
