import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import React, { useEffect } from "react";

import { realtimeApi } from "../api/endpoints";
import { absoluteUrl } from "../api/client";
import type { Message, MessagePage, RealtimeEvent } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { useFeedback } from "../ui/Feedback";

export const keys = {
  profile: ["profile"] as const,
  discover: ["discover"] as const,
  matches: ["matches"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  summary: ["summary"] as const,
  blocked: ["blocked"] as const,
  interests: ["interests"] as const,
};

/** Merges a message into the cached pages of a conversation (newest page first), ignoring duplicates. */
export function mergeMessage(data: InfiniteData<MessagePage> | undefined, message: Message) {
  if (!data) return data;
  if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) return data;
  const [first, ...rest] = data.pages;
  if (!first) return data;
  return { ...data, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
}

export function markMessagesRead(
  data: InfiniteData<MessagePage> | undefined,
  readerIsOther: boolean,
  myId: string,
) {
  if (!data) return data;
  const now = new Date().toISOString();
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      messages: p.messages.map((m) =>
        readerIsOther === (m.senderId === myId) && !m.readAt ? { ...m, readAt: now } : m,
      ),
    })),
  };
}

export function applyEvent(qc: QueryClient, event: RealtimeEvent, myId: string): void {
  switch (event.type) {
    case "message":
      qc.setQueryData<InfiniteData<MessagePage>>(keys.messages(event.payload.matchId), (d) =>
        mergeMessage(d, event.payload),
      );
      void qc.invalidateQueries({ queryKey: keys.matches });
      void qc.invalidateQueries({ queryKey: keys.summary });
      void qc.invalidateQueries({ queryKey: keys.notifications });
      break;
    case "match":
      void qc.invalidateQueries({ queryKey: keys.matches });
      void qc.invalidateQueries({ queryKey: keys.summary });
      void qc.invalidateQueries({ queryKey: keys.notifications });
      break;
    case "read":
      // The other person read my messages.
      qc.setQueryData<InfiniteData<MessagePage>>(keys.messages(event.payload.matchId), (d) =>
        markMessagesRead(d, true, myId),
      );
      break;
    case "unmatched":
      qc.removeQueries({ queryKey: keys.messages(event.payload.matchId) });
      void qc.invalidateQueries({ queryKey: keys.matches });
      void qc.invalidateQueries({ queryKey: keys.summary });
      break;
  }
}

/**
 * Keeps one WebSocket open while signed in. Commands go through REST; the socket only pushes
 * events, so a dropped connection never loses data: on reconnect everything is refetched.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();
  const { status, user } = useAuth();
  const { toast } = useFeedback();
  const userId = user?.id;

  useEffect(() => {
    if (status !== "authenticated" || !userId) return;
    let socket: WebSocket | null = null;
    let retry = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const connect = async () => {
      try {
        const { ticket } = await realtimeApi.ticket();
        if (stopped) return;
        const url = absoluteUrl(`/realtime/ws?ticket=${encodeURIComponent(ticket)}`).replace(
          /^http/,
          "ws",
        );
        socket = new WebSocket(url);
        socket.onopen = () => {
          if (retry > 0) void qc.invalidateQueries(); // resync after an outage
          retry = 0;
        };
        socket.onmessage = (e) => {
          try {
            const event = JSON.parse(String(e.data)) as RealtimeEvent;
            applyEvent(qc, event, userId);
            if (event.type === "match") toast("Nouveau match !", "success");
          } catch {
            // ignore malformed frames
          }
        };
        socket.onclose = schedule;
        socket.onerror = () => socket?.close();
      } catch {
        schedule();
      }
    };

    const schedule = () => {
      if (stopped) return;
      retry += 1;
      timer = setTimeout(connect, Math.min(30_000, 1000 * 2 ** Math.min(retry, 5)));
    };

    void connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      if (socket) {
        socket.onclose = null;
        socket.close();
      }
    };
  }, [status, userId, qc, toast]);

  return <>{children}</>;
}
