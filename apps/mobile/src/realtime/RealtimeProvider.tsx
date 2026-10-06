import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL, getFreshAccessToken } from "../api/client";
import type { ServerEvent } from "../api/types";
import { useAuth } from "../hooks/useAuth";

type Listener = (event: ServerEvent) => void;

type RealtimeContextValue = {
  subscribe: (listener: Listener) => () => void;
};

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

const MAX_BACKOFF_MS = 30_000;

function websocketUrl(): string | null {
  if (!API_BASE_URL) {
    return null;
  }
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}

function parseEvent(raw: unknown): ServerEvent | null {
  if (typeof raw !== "string") {
    return null;
  }
  try {
    const value = JSON.parse(raw) as { type?: unknown };
    return typeof value.type === "string" ? (value as ServerEvent) : null;
  } catch {
    return null;
  }
}

/**
 * Keeps one authenticated WebSocket open while the user is signed in and the
 * app is in the foreground, reconnecting with exponential backoff. Events are
 * fanned out to subscribers and keep the shared query caches fresh.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const queryClient = useQueryClient();
  const listeners = useRef(new Set<Listener>());

  const subscribe = useCallback((listener: Listener) => {
    listeners.current.add(listener);
    return () => {
      listeners.current.delete(listener);
    };
  }, []);

  useEffect(() => {
    if (status !== "signedIn") {
      return;
    }
    const url = websocketUrl();
    if (!url) {
      return;
    }

    let socket: WebSocket | null = null;
    let retry = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const handle = (event: ServerEvent) => {
      switch (event.type) {
        case "message.new":
        case "message.read":
        case "match.new":
        case "match.ended":
          void queryClient.invalidateQueries({ queryKey: ["matches"] });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          break;
        default:
          break;
      }
      listeners.current.forEach((listener) => listener(event));
    };

    const connect = async () => {
      if (stopped || AppState.currentState !== "active") {
        return;
      }
      const token = await getFreshAccessToken();
      if (!token || stopped) {
        return;
      }
      const ws = new WebSocket(url);
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (message) => {
        const event = parseEvent(message.data);
        if (!event) {
          return;
        }
        if (event.type === "ready") {
          if (retry > 0) {
            // We may have missed events while disconnected: re-sync lists.
            void queryClient.invalidateQueries({ queryKey: ["matches"] });
            void queryClient.invalidateQueries({ queryKey: ["notifications"] });
            listeners.current.forEach((listener) => listener({ type: "resync" }));
          }
          retry = 0;
        }
        handle(event);
      };
      ws.onclose = () => {
        if (socket === ws) {
          socket = null;
        }
        if (stopped) {
          return;
        }
        const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** retry);
        retry += 1;
        timer = setTimeout(() => {
          timer = null;
          void connect();
        }, delay);
      };
      ws.onerror = () => ws.close();
    };

    const appStateSub = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        if (!socket && !timer) {
          void connect();
        }
      } else if (socket) {
        socket.close(); // do not hold a socket in the background
      }
    });

    void connect();

    return () => {
      stopped = true;
      appStateSub.remove();
      if (timer) {
        clearTimeout(timer);
      }
      socket?.close();
    };
  }, [queryClient, status]);

  const value = useMemo(() => ({ subscribe }), [subscribe]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

/** Runs `handler` for every server event while the component is mounted. */
export function useRealtimeEvents(handler: Listener): void {
  const context = useContext(RealtimeContext);
  const latest = useRef(handler);
  latest.current = handler;
  useEffect(() => {
    if (!context) {
      return;
    }
    return context.subscribe((event) => latest.current(event));
  }, [context]);
}
