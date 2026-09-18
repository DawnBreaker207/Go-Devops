import { useEffect, useState } from 'react';

const remaining = (deadline: string | undefined): number => {
  if (!deadline) return 0;
  // Khong cat chuoi, khong so sanh chuoi: cung mot thoi diem tu backend nay ve
  // voi hai offset khac nhau (+07:00 o endpoint nay, Z o endpoint kia).
  // Date.parse xu ly dung ca hai.
  const ms = Date.parse(deadline) - Date.now();
  return ms > 0 ? Math.floor(ms / 1000) : 0;
};

/**
 * So GIAY con lai toi mot moc thoi gian, dem lui moi giay. Tra ve 0 khi da qua
 * moc va khong dem am.
 *
 * Gia tri duoc TINH LAI MOI LAN RENDER thay vi giu trong state: nhip dap chi
 * lam component render lai. Nho vay doi `deadline` la co gia tri moi ngay lap
 * tuc, khong phai doi het mot giay - va khong phai goi setState dong bo trong
 * effect, thu ma react-hooks/set-state-in-effect chan (dung: no gay chuoi
 * render thua).
 */
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
