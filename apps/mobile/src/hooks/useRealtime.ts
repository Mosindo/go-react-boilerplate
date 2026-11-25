import { useEffect, useSyncExternalStore } from "react";
import type { InfiniteData } from "@tanstack/react-query";
import { realtime, type RealtimeEvent } from "../api/realtime";
import type { MessagesPage } from "../api/types";
import { showToast } from "../ui/Toast";
import { queryClient, queryKeys } from "./queryClient";
import { insertMessage } from "./cache";

/** Whether the realtime socket is currently connected. */
export function useRealtimeConnected(): boolean {
  return useSyncExternalStore(
    (l) => realtime.onStatus(l),
    () => realtime.connected,
    () => false
  );
}

let activeConversationId: string | null = null;
const celebratedMatches = new Set<string>();

/** Marks a match already shown in the match modal so no toast duplicates it. */
export function markMatchCelebrated(conversationId: string) {
  celebratedMatches.add(conversationId);
}

/** Lets the chat screen mark which conversation is on screen. */
export function setActiveConversation(id: string | null) {
  activeConversationId = id;
}

export function applyRealtimeEvent(event: RealtimeEvent, currentUserId: string) {
  switch (event.type) {
    case "message.created": {
      const msg = event.data;
      queryClient.setQueryData<InfiniteData<MessagesPage, string | null>>(queryKeys.messages(msg.conversationId), (d) =>
        insertMessage(d, msg)
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      if (msg.senderId !== currentUserId && activeConversationId !== msg.conversationId) {
        showToast("Nouveau message reçu", "info");
      }
      break;
    }
    case "conversation.read": {
      queryClient.setQueryData<InfiniteData<MessagesPage, string | null>>(queryKeys.messages(event.data.conversationId), (d) =>
        d ? { ...d, pages: d.pages.map((p, i) => (i === 0 ? { ...p, peerLastReadAt: event.data.readAt } : p)) } : d
      );
      break;
    }
    case "match.created":
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      break;
    case "match.ended":
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.chatUnread });
      break;
    case "notification.created":
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
      if (event.data.type === "match" && !celebratedMatches.has(event.data.data.conversationId ?? "")) {
        showToast(event.data.body, "success");
      }
      break;
    case "ready":
      // Catch up on anything missed while disconnected.
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
      break;
  }
}

/** Starts the realtime connection while signed in and syncs caches. */
export function useRealtimeSync(enabled: boolean, currentUserId: string | undefined) {
  useEffect(() => {
    if (!enabled || !currentUserId) {
      return;
    }
    realtime.start();
    const unsubscribe = realtime.subscribe((event) => {
      applyRealtimeEvent(event, currentUserId);
      if (event.type === "message.created") {
        void queryClient.invalidateQueries({ queryKey: queryKeys.chatUnread });
      }
    });
    return () => {
      unsubscribe();
      realtime.stop();
    };
  }, [enabled, currentUserId]);
}
