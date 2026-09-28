import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { orderApi } from '@/api/order.api';
import type { SeatType } from '@/types';

export interface FlowSeat {
  seatId: string;
  label: string;
  seatType: SeatType;
  price: number;
}

export const MAX_FLOW_SEATS = 8;

interface BookingFlowState {
  showtimeId: string | null;
  bookingId: string | null;
  seats: FlowSeat[];
  selectedIds: string[];
  setShowtime: (showtimeId: string) => void;
  enterShowtime: (showtimeId: string) => void;
  setBooking: (bookingId: string, seats: FlowSeat[]) => void;
  adopt: (showtimeId: string, bookingId: string, seats: FlowSeat[]) => void;
  toggleSeat: (seatId: string) => void;
  clearSelection: () => void;
  clearBooking: () => void;
  clear: () => void;
}

export const useBookingFlowStore = create<BookingFlowState>()(
  persist(
    (set, get) => ({
      showtimeId: null,
      bookingId: null,
      seats: [],
      selectedIds: [],

      setShowtime: (showtimeId) => set({ showtimeId, bookingId: null, seats: [], selectedIds: [] }),
      enterShowtime: (showtimeId) => {
        if (get().showtimeId === showtimeId) return;
        set({ showtimeId, bookingId: null, seats: [], selectedIds: [] });
      },
      setBooking: (bookingId, seats) => set({ bookingId, seats }),
      adopt: (showtimeId, bookingId, seats) =>
        set({ showtimeId, bookingId, seats, selectedIds: [] }),
      toggleSeat: (seatId) => {
        const { selectedIds, bookingId } = get();
        const next = new Set(selectedIds);
        if (next.has(seatId)) next.delete(seatId);
        else if (next.size < MAX_FLOW_SEATS) next.add(seatId);
        if (next.size === 0 && bookingId) {
          set({ selectedIds: [], bookingId: null, seats: [] });
          void orderApi.cancel(bookingId).catch(() => undefined);
          return;
        }
        set({ selectedIds: [...next] });
      },
      clearSelection: () => set({ selectedIds: [] }),
      clearBooking: () => {
        const { bookingId } = get();
        set({ bookingId: null, seats: [] });
        if (bookingId) void orderApi.cancel(bookingId).catch(() => undefined);
      },
      clear: () => set({ showtimeId: null, bookingId: null, seats: [], selectedIds: [] }),
    }),
    {
      name: 'cp-booking-flow',
      storage: createJSONStorage(() => sessionStorage),
    }
  )
);
