import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { realtimeUrl } from "../api/client";
import { realtimeApi } from "../api/endpoints";
import type { Match, Message, Page, RealtimeEvent } from "../api/types";
import { insertMessage, keys, markMessagesRead, removeMatch } from "./queries";

// --- tiny store for "it's a match" celebrations and connection state ---------
let celebration: Match | null = null;
let connected = false;
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((l) => l());
const subscribe = (l: () => void) => {
  listeners.add(l);
  return () => listeners.delete(l);
};

export function celebrateMatch(match: Match | null) {
  celebration = match;
  emit();
}
export function useCelebration() {
  return useSyncExternalStore(
    subscribe,
    () => celebration,
    () => celebration
  );
}
export function useRealtimeConnected() {
  return useSyncExternalStore(
    subscribe,
    () => connected,
    () => connected
  );
}
function setConnected(value: boolean) {
  if (connected !== value) {
    connected = value;
    emit();
  }
}

/**
 * Keeps a WebSocket open while the app is in the foreground and applies
 * server events to the React Query cache. On every (re)connection the
 * relevant queries are refetched, so missed events are never lost.
 */
export function useRealtime(enabled: boolean, currentUserId: string | null) {
  const qc = useQueryClient();
  const [appActive, setAppActive] = useState(AppState.currentState !== "background");
  const userRef = useRef(currentUserId);
  userRef.current = currentUserId;

  useEffect(() => {
    const sub = AppState.addEventListener("change", (s) => setAppActive(s === "active"));
    return () => sub.remove();
  }, []);

  useEffect(() => {
    if (!enabled || !appActive) return;
    let socket: WebSocket | null = null;
    let retry: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let stopped = false;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new": {
          const m = event.data;
          qc.setQueryData<InfiniteData<Page<Message>, string | null>>(keys.messages(m.matchId), (d) => insertMessage(d, m));
          void qc.invalidateQueries({ queryKey: keys.matches });
          break;
        }
        case "message.read":
          qc.setQueryData<InfiniteData<Page<Message>, string | null>>(keys.messages(event.data.matchId), (d) =>
            markMessagesRead(d, event.data.readerId, event.data.readAt)
          );
          break;
        case "match.new":
          void qc.invalidateQueries({ queryKey: keys.matches });
          celebrateMatch(event.data);
          break;
        case "match.removed":
          qc.setQueryData<InfiniteData<Page<Match>, string | null>>(keys.matches, (d) => removeMatch(d, event.data.matchId));
          qc.removeQueries({ queryKey: keys.messages(event.data.matchId) });
          break;
        case "notification.new":
          void qc.invalidateQueries({ queryKey: keys.notifications });
          break;
      }
    };

    const connect = async () => {
      try {
        const ticket = await realtimeApi.ticket();
        if (stopped) return;
        socket = new WebSocket(realtimeUrl(ticket));
        socket.onopen = () => {
          attempt = 0;
          setConnected(true);
          void qc.invalidateQueries({ queryKey: keys.matches });
          void qc.invalidateQueries({ queryKey: keys.notifications });
          void qc.invalidateQueries({ queryKey: ["messages"] });
        };
        socket.onmessage = (msg) => {
          try {
            handle(JSON.parse(String(msg.data)) as RealtimeEvent);
          } catch {
            /* ignore malformed frames */
          }
        };
        socket.onclose = () => {
          setConnected(false);
          schedule();
        };
        socket.onerror = () => socket?.close();
      } catch {
        schedule();
      }
    };

    const schedule = () => {
      if (stopped) return;
      attempt += 1;
      const delay = Math.min(30_000, 1000 * 2 ** Math.min(attempt, 5)) + Math.random() * 500;
      retry = setTimeout(connect, delay);
    };

    void connect();
    return () => {
      stopped = true;
      if (retry) clearTimeout(retry);
      socket?.close();
      setConnected(false);
    };
  }, [enabled, appActive, qc]);
}
