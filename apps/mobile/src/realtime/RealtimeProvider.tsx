import React, { createContext, useContext, useEffect, useMemo, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { realtimeApi } from "../api/services";
import type { MessagesPage, RealtimeEvent } from "../api/types";
import { useToast } from "../components/Toast";
import { fr } from "../i18n/fr";

type RealtimeContextValue = {
  /** The conversation currently on screen; incoming messages there are marked read. */
  setActiveConversation: (id: string | null) => void;
  onMessageInActive: (listener: (conversationId: string) => void) => () => void;
};

const RealtimeContext = createContext<RealtimeContextValue>({
  setActiveConversation: () => undefined,
  onMessageInActive: () => () => undefined
});

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}

function toWsUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws?ticket=${encodeURIComponent(ticket)}`;
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const active = useRef<string | null>(null);
  const listeners = useRef(new Set<(id: string) => void>());

  useEffect(() => {
    let stopped = false;
    let socket: WebSocket | null = null;
    let retry = 1000;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new": {
          const message = event.data;
          queryClient.setQueryData<InfiniteData<MessagesPage>>(["messages", message.conversationId], (old) => {
            if (!old || old.pages.length === 0) return old;
            const [first, ...rest] = old.pages;
            if (!first || first.messages.some((m) => m.id === message.id)) return old;
            return { ...old, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
          });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          if (active.current === message.conversationId) listeners.current.forEach((l) => l(message.conversationId));
          break;
        }
        case "message.read":
          void queryClient.invalidateQueries({ queryKey: ["messages", event.data.conversationId] });
          break;
        case "match.new":
          toast.show(fr.discover.matchBody(event.data.firstName), "success");
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          break;
        case "match.removed":
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          break;
      }
    };

    const connect = async () => {
      if (stopped) return;
      try {
        const ticket = await realtimeApi.ticket();
        if (stopped) return;
        const ws = new WebSocket(toWsUrl(ticket));
        socket = ws;
        ws.onopen = () => {
          retry = 1000;
          // Anything missed while offline is recovered with a refetch.
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
        };
        ws.onmessage = (e) => {
          try {
            handle(JSON.parse(String(e.data)) as RealtimeEvent);
          } catch {
            // Ignore malformed frames.
          }
        };
        ws.onclose = () => scheduleReconnect();
        ws.onerror = () => ws.close();
      } catch {
        scheduleReconnect();
      }
    };

    const scheduleReconnect = () => {
      if (stopped || timer) return;
      timer = setTimeout(() => {
        timer = null;
        void connect();
      }, retry);
      retry = Math.min(retry * 2, 30000);
    };

    void connect();
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active" && (!socket || socket.readyState > 1)) void connect();
    });

    return () => {
      stopped = true;
      sub.remove();
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [queryClient, toast]);

  const value = useMemo<RealtimeContextValue>(
    () => ({
      setActiveConversation: (id) => {
        active.current = id;
      },
      onMessageInActive: (listener) => {
        listeners.current.add(listener);
        return () => listeners.current.delete(listener);
      }
    }),
    []
  );

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}
