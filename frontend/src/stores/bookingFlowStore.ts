import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { orderApi } from '@/api/order.api';
import type { SeatType } from '@/types';

/** One seat in the order for review (id + label + locked price). */
export interface FlowSeat {
  seatId: string;
  label: string;
  seatType: SeatType;
  price: number;
}

/** Backend cap per hold. */
export const MAX_FLOW_SEATS = 8;

/** Shared booking-flow state (parent owns, 3 steps read/write) replacing location.state (lost on F5). Minimal: showtime, booking, seats, draft picks. Truth (order, movie, countdown) stays in react-query. Persisted to sessionStorage. Picks live in the store so `toggle` keeps a stable reference for memo SeatButton. */
interface BookingFlowState {
  showtimeId: string | null;
  bookingId: string | null;
  seats: FlowSeat[];
  selectedIds: string[];
  setShowtime: (showtimeId: string) => void;
  /** Same showtime keeps draft picks (late init must not wipe just-clicked seats). */
  enterShowtime: (showtimeId: string) => void;
  setBooking: (bookingId: string, seats: FlowSeat[]) => void;
  /** Adopt an order from elsewhere (e.g. "View order" in My tickets) to continue in-flow. */
  adopt: (showtimeId: string, bookingId: string, seats: FlowSeat[]) => void;
  toggleSeat: (seatId: string) => void;
  clearSelection: () => void;
  /** Server-side cancel (frees seats) then drop booking. Skipped for natural expiry (sweeper cleans up). Cancel errors still clear local state. */
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
        // Deselecting everything cancels the hold inline (not in an effect). Cancel errors still clear local state.
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
        // Cancel the old order server-side to free seats now (don't wait for the 10' sweep). Errors ignored (may already be expired); local is already cleared.
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
