import { useEffect, useRef, useState } from 'react';
import { apiClient, unwrap, API_BASE_URL } from '@/api/client';
import type { ApiResponse, SeatStatus } from '@/types';

/** One seat's status change, as the backend emits in "seats" events. */
export interface SeatStatusUpdate {
  /** showtime_seats.id - matches SeatMapSeat.showtime_seat_id here. */
  id: string;
  status: SeatStatus;
}

interface SeatsFramePayload {
  showtime_id: string;
  hall_id: string;
  seats: SeatStatusUpdate[];
}

interface RealtimeTokenResponse {
  token: string;
  expires_in: number;
  stream_url: string;
}

/** Reconnects always mint via /events/token (30s token, stream self-closes at maxAge). Short delay suffices - the server already sends "retry: 3000" and debounces per seat batch. */
const RECONNECT_DELAY_MS = 5000;

/** Live seat states over SSE: mint a short token first (EventSource sends no auth), stream debounced `seats` events, re-mint on `onerror`; returns `sseDown` for fallback. */
export function useSeatMapRealtime(
  showtimeId: string | undefined,
  enabled: boolean,
  onSeats: (seats: SeatStatusUpdate[]) => void
): { sseDown: boolean } {
  const [sseDown, setSseDown] = useState(false);
  // Ref always holds the latest callback (EventSource closures go stale).
  // Assigned in an effect (post-render), never during render.
  const onSeatsRef = useRef(onSeats);
  useEffect(() => {
    onSeatsRef.current = onSeats;
  });

  useEffect(() => {
    if (!enabled || !showtimeId) {
      return;
    }

    let cancelled = false;
    let source: EventSource | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const cleanupSource = () => {
      source?.close();
      source = null;
    };

    const scheduleReconnect = () => {
      if (cancelled) return;
      setSseDown(true);
      retryTimer = setTimeout(() => void connect(), RECONNECT_DELAY_MS);
    };

    const connect = async () => {
      try {
        const response = await apiClient.get<ApiResponse<RealtimeTokenResponse>>('/events/token', {
          params: { show_id: showtimeId },
        });
        if (cancelled) return;
        const { token } = unwrap(response);

        const es = new EventSource(
          `${API_BASE_URL}/events/shows/${showtimeId}?token=${encodeURIComponent(token)}`
        );
        source = es;

        es.addEventListener('seats', (event) => {
          try {
            const payload = JSON.parse((event as MessageEvent<string>).data) as SeatsFramePayload;
            setSseDown(false);
            onSeatsRef.current(payload.seats);
          } catch {
            // Malformed frame: skip, the next debounced batch resends.
          }
        });
        es.onopen = () => setSseDown(false);
        es.onerror = () => {
          cleanupSource();
          scheduleReconnect();
        };
      } catch {
        // Token mint failed (offline, 401, closed showtime...): retry later.
        scheduleReconnect();
      }
    };

    void connect();

    return () => {
      cancelled = true;
      if (retryTimer) clearTimeout(retryTimer);
      cleanupSource();
    };
  }, [showtimeId, enabled]);

  // Realtime off (no showtimeId or enabled=false) always reports "not down", never stale state from a previous run.
  return { sseDown: enabled && showtimeId ? sseDown : false };
}
