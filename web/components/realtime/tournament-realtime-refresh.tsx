"use client";

import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef } from "react";

type RealtimeEvent = {
  type: string;
  tournamentId: number;
  occurredAt: string;
};

const DEFAULT_EVENT_TYPES = ["TOURNAMENT_UPDATED", "MATCH_UPDATED", "SCHEDULE_UPDATED", "TEAM_UPDATED"];

function buildWsURL(tournamentId: number): string | null {
  const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  try {
    const url = new URL(base);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = "/ws";
    url.search = `tournamentId=${tournamentId}`;
    return url.toString();
  } catch {
    return null;
  }
}

export function TournamentRealtimeRefresh({ tournamentId, refreshIntervalMs = 800 }: { tournamentId: number; refreshIntervalMs?: number }) {
  const router = useRouter();
  const wsURL = useMemo(() => buildWsURL(tournamentId), [tournamentId]);
  const lastRefreshAt = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!wsURL) {
      return;
    }

    let isDisposed = false;
    let socket: WebSocket | null = null;

    const connect = () => {
      if (isDisposed) return;

      socket = new WebSocket(wsURL);
      socket.onmessage = (messageEvent) => {
        try {
          const payload = JSON.parse(messageEvent.data) as RealtimeEvent;
          if (!DEFAULT_EVENT_TYPES.includes(payload.type)) {
            return;
          }
          const now = Date.now();
          if (now-lastRefreshAt.current < refreshIntervalMs) {
            return;
          }
          lastRefreshAt.current = now;
          router.refresh();
        } catch {
          // ignore malformed events
        }
      };
      socket.onclose = () => {
        if (isDisposed) return;
        reconnectTimer.current = setTimeout(connect, 1500);
      };
      socket.onerror = () => {
        socket?.close();
      };
    };

    connect();

    return () => {
      isDisposed = true;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
      socket?.close();
    };
  }, [wsURL, router, refreshIntervalMs]);

  return null;
}
