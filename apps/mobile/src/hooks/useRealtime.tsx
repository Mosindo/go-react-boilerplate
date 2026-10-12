import { type InfiniteData, useQueryClient } from "@tanstack/react-query";
import React, { createContext, useContext, useEffect, useMemo, useRef } from "react";
import { AppState } from "react-native";
import { me } from "../api/auth";
import { wsUrl } from "../api/client";
import type { ChatMessage, MessagePage, ServerEvent } from "../api/types";
import { getTokens } from "../store/tokenStore";
import { useAuth } from "./useAuth";
import { keys } from "./queries";

type Listener = (event: ServerEvent) => void;

type RealtimeApi = {
  /** Subscribe to raw server events (e.g. to show the "It's a match" screen). */
  subscribe: (listener: Listener) => () => void;
};

const RealtimeContext = createContext<RealtimeApi>({ subscribe: () => () => undefined });

type MessagesCache = InfiniteData<MessagePage, string | undefined>;

export function appendMessage(cache: MessagesCache | undefined, message: ChatMessage): MessagesCache | undefined {
  if (!cache || cache.pages.length === 0) {
    return cache;
  }
  const exists = cache.pages.some((page) => page.items.some((item) => item.id === message.id));
  if (exists) {
    return cache;
  }
  const [first, ...rest] = cache.pages;
  return { ...cache, pages: [{ ...first, items: [message, ...first.items] }, ...rest] };
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const client = useQueryClient();
  const listeners = useRef(new Set<Listener>());

  useEffect(() => {
    if (!user) {
      return;
    }
    let socket: WebSocket | null = null;
    let retry = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let closedByUs = false;

    const dispatch = (event: ServerEvent) => {
      switch (event.type) {
        case "message.new":
          client.setQueryData<MessagesCache>(keys.messages(event.data.matchId), (cache) => appendMessage(cache, event.data));
          void client.invalidateQueries({ queryKey: keys.matches });
          break;
        case "message.read":
          client.setQueryData<MessagesCache>(keys.messages(event.data.matchId), (cache) =>
            cache && cache.pages.length > 0
              ? { ...cache, pages: [{ ...cache.pages[0], otherLastReadAt: event.data.readAt }, ...cache.pages.slice(1)] }
              : cache
          );
          break;
        case "match.new":
          void client.invalidateQueries({ queryKey: keys.matches });
          void client.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "match.removed":
          client.removeQueries({ queryKey: keys.messages(event.data.matchId) });
          void client.invalidateQueries({ queryKey: keys.matches });
          void client.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "notification.new":
          void client.invalidateQueries({ queryKey: keys.notifications });
          break;
        default:
          break;
      }
      listeners.current.forEach((listener) => listener(event));
    };

    const connect = async () => {
      if (closedByUs) {
        return;
      }
      const tokens = await getTokens();
      if (!tokens) {
        return;
      }
      socket = new WebSocket(wsUrl());
      socket.onopen = () => {
        socket?.send(JSON.stringify({ type: "auth", token: tokens.accessToken }));
      };
      socket.onmessage = (message) => {
        try {
          const event = JSON.parse(String(message.data)) as ServerEvent;
          if (event.type === "ready") {
            // After (re)connecting, anything may have changed while we were away.
            if (retry > 0) {
              void client.invalidateQueries();
            }
            retry = 0;
          }
          dispatch(event);
        } catch {
          // ignore malformed frames
        }
      };
      socket.onclose = () => {
        socket = null;
        if (closedByUs) {
          return;
        }
        retry += 1;
        const delay = Math.min(30_000, 1000 * 2 ** Math.min(retry, 5));
        timer = setTimeout(() => {
          // A cheap authenticated call refreshes an expired access token before reconnecting.
          me()
            .catch(() => undefined)
            .finally(() => void connect());
        }, delay);
      };
      socket.onerror = () => socket?.close();
    };

    void connect();

    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active" && !socket && !timer) {
        void connect();
      }
    });

    return () => {
      closedByUs = true;
      appState.remove();
      if (timer) {
        clearTimeout(timer);
      }
      socket?.close();
    };
  }, [client, user]);

  const api = useMemo<RealtimeApi>(
    () => ({
      subscribe: (listener) => {
        listeners.current.add(listener);
        return () => {
          listeners.current.delete(listener);
        };
      }
    }),
    []
  );

  return <RealtimeContext.Provider value={api}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeApi {
  return useContext(RealtimeContext);
}
