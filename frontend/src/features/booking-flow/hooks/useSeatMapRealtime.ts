import { useEffect, useRef, useState } from 'react';
import { apiClient, unwrap, API_BASE_URL } from '@/api/client';
import type { ApiResponse, SeatStatus } from '@/types';

export interface SeatStatusUpdate {
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

// Reconnects always mint fresh via /events/token: EventSource sends no auth.
const RECONNECT_DELAY_MS = 5000;

// Live seats over SSE; on realtime loss report sseDown for the poll fallback.
export function useSeatMapRealtime(
  showtimeId: string | undefined,
  enabled: boolean,
  onSeats: (seats: SeatStatusUpdate[]) => void
): { sseDown: boolean } {
  const [sseDown, setSseDown] = useState(false);
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
            // A broken frame is skipped: the next batch resends.
          }
        });
        es.onopen = () => setSseDown(false);
        es.onerror = () => {
          cleanupSource();
          scheduleReconnect();
        };
      } catch {
        // A failed token mint just schedules the reconnect.
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

  // While realtime is off, always report up: avoids stale state from a previous run.
  return { sseDown: enabled && showtimeId ? sseDown : false };
}
