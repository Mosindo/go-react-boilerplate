import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import { realtime, type RealtimeEvent } from "../api/realtime";
import type { MessagesPage } from "../api/types";
import { useSession } from "../session/SessionProvider";
import { toast } from "../ui/Toast";

const ConnectedContext = createContext(false);

/** Match ids already celebrated on screen (to avoid a duplicate toast). */
export const celebratedMatches = new Set<string>();

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useSession();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (status !== "signedIn") {
      realtime.stop();
      return;
    }
    realtime.start();
    const offStatus = realtime.onStatus((value) => {
      setConnected(value);
      // After a reconnect, re-sync anything we may have missed.
      if (value) void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
    });
    const offEvents = realtime.subscribe((event) => handleEvent(event));
    const appState = AppState.addEventListener("change", (next) => {
      if (next === "active") realtime.poke();
    });
    return () => {
      offStatus();
      offEvents();
      appState.remove();
      realtime.stop();
    };

    function handleEvent(event: RealtimeEvent) {
      switch (event.type) {
        case "message.new": {
          const msg = event.data;
          queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(msg.conversationId), (data) => {
            if (!data || data.pages.length === 0) return data;
            if (data.pages.some((p) => p.messages.some((m) => m.id === msg.id))) return data;
            const [first, ...rest] = data.pages;
            return { ...data, pages: [{ ...first, messages: [msg, ...first.messages] }, ...rest] };
          });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
          break;
        }
        case "conversation.read": {
          const { conversationId, readAt } = event.data;
          queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) =>
            data ? { ...data, pages: data.pages.map((p) => ({ ...p, otherLastReadAt: readAt })) } : data
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        }
        case "match.new": {
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
          void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
          if (!celebratedMatches.has(event.data.id)) {
            celebratedMatches.add(event.data.id);
            toast(`Nouveau match avec ${event.data.user.firstName} !`, "success");
          }
          break;
        }
        case "match.removed":
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
          if (event.data.type === "like") void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
          break;
        default:
          break;
      }
    }
  }, [status, queryClient]);

  return <ConnectedContext.Provider value={connected}>{children}</ConnectedContext.Provider>;
}

/** True while the realtime socket is up; screens poll as a fallback otherwise. */
export function useRealtimeConnected(): boolean {
  return useContext(ConnectedContext);
}

/** Subscribe a screen to realtime events. */
export function useRealtimeEvent(handler: (event: RealtimeEvent) => void) {
  const ref = useRef(handler);
  ref.current = handler;
  useEffect(() => realtime.subscribe((event) => ref.current(event)), []);
}
