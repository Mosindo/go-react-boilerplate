import React, { createContext, useContext, useEffect, useMemo, useRef } from "react";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { websocketUrl } from "../api/client";
import { currentTokens } from "../api/tokenStore";
import type { MessagePage, RealtimeEvent } from "../api/types";
import { useToast } from "../ui/Toast";
import { keys } from "./keys";
import { useSession } from "./session";

type RealtimeValue = {
  /** The chat screen declares which conversation is on screen to avoid redundant alerts. */
  setActiveConversation: (id: string | null) => void;
};

const RealtimeContext = createContext<RealtimeValue>({ setActiveConversation: () => undefined });

const MAX_BACKOFF_MS = 30_000;

/** Merge a pushed message into the cached, newest-first, paginated message list. */
export function mergeMessage(
  data: InfiniteData<MessagePage> | undefined,
  message: { id: number }
): InfiniteData<MessagePage> | undefined {
  if (!data || data.pages.length === 0) return data;
  const exists = data.pages.some((page) => page.messages.some((m) => m.id === message.id));
  if (exists) return data;
  const [first, ...rest] = data.pages;
  return {
    ...data,
    pages: [
      { ...first, messages: [message as MessagePage["messages"][number], ...first.messages] },
      ...rest
    ]
  };
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { user } = useSession();
  const activeConversation = useRef<string | null>(null);
  const userId = user?.id;

  useEffect(() => {
    if (!userId) return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let stopped = false;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "ready":
          attempt = 0;
          break;
        case "message": {
          const msg = event.payload;
          queryClient.setQueryData<InfiniteData<MessagePage>>(
            keys.messages(msg.conversationId),
            (d) => mergeMessage(d, msg)
          );
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          if (msg.senderId !== userId && activeConversation.current !== msg.conversationId) {
            toast.show("New message", "info");
          }
          break;
        }
        case "read":
          void queryClient.invalidateQueries({
            queryKey: keys.messages(event.payload.conversationId)
          });
          break;
        case "match":
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          toast.show("It's a match!", "success");
          break;
      }
    };

    const schedule = () => {
      if (stopped) return;
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt);
      attempt += 1;
      timer = setTimeout(connect, delay);
    };

    const connect = () => {
      const token = currentTokens()?.accessToken;
      if (stopped || !token) return;
      const ws = new WebSocket(websocketUrl());
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (e) => {
        try {
          handle(JSON.parse(String(e.data)) as RealtimeEvent);
        } catch {
          // Ignore malformed frames.
        }
      };
      ws.onclose = () => {
        socket = null;
        if (stopped) return;
        // An expired access token is refreshed by any authenticated REST call.
        void authApi
          .me()
          .catch(() => undefined)
          .finally(schedule);
        // Anything missed while offline is recovered from the REST API.
        void queryClient.invalidateQueries({ queryKey: keys.conversations });
      };
      ws.onerror = () => ws.close();
    };

    connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [userId, queryClient, toast]);

  const value = useMemo<RealtimeValue>(
    () => ({
      setActiveConversation: (id) => {
        activeConversation.current = id;
      }
    }),
    []
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}
