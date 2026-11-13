import { useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { RealtimeClient, type RealtimeEvent } from "../api/realtime";
import type { Conversation, MatchInfo, Message } from "../api/types";
import { useToast } from "../ui";
import { appendMessage, type MessagesData } from "./queries";
import { queryKeys } from "./queryClient";
import { useSession } from "./session";

// Likes sent from the swipe screen, which shows its own match modal. The
// realtime event can arrive before the HTTP response, so the target is
// registered before the request and forgotten after a short delay.
const pendingLikes = new Set<string>();
export function expectMatchFrom(userId: string) {
  pendingLikes.add(userId);
  setTimeout(() => pendingLikes.delete(userId), 15_000);
}

/** Keeps the React Query cache in sync with server-pushed events. */
export function RealtimeBridge() {
  const { status, userId } = useSession();
  const client = useQueryClient();
  const toast = useToast();
  const toastRef = useRef(toast);
  toastRef.current = toast;

  useEffect(() => {
    if (status !== "signedIn" || !userId) {
      return;
    }
    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.created": {
          const message = event.data as Message;
          const peerId = message.senderUserId === userId ? message.recipientUserId : message.senderUserId;
          client.setQueryData<MessagesData>(queryKeys.messages(peerId), (data) => appendMessage(data, message));
          client.setQueryData<Conversation[]>(queryKeys.conversations, (list) => {
            if (!list) return list;
            const index = list.findIndex((c) => c.user.userId === peerId);
            if (index < 0) return list;
            const updated: Conversation = {
              ...list[index],
              lastMessage: message,
              unreadCount: message.senderUserId === userId ? list[index].unreadCount : list[index].unreadCount + 1
            };
            return [updated, ...list.slice(0, index), ...list.slice(index + 1)];
          });
          break;
        }
        case "conversation.read": {
          const data = event.data as { userId: string; readAt: string };
          client.setQueryData<MessagesData>(queryKeys.messages(data.userId), (cache) =>
            cache
              ? { ...cache, pages: cache.pages.map((p, i) => (i === 0 ? { ...p, peerLastReadAt: data.readAt } : p)) }
              : cache
          );
          break;
        }
        case "match.created": {
          const match = event.data as MatchInfo;
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          if (!pendingLikes.has(match.user.userId)) {
            toastRef.current.show(`Nouveau match avec ${match.user.firstName} !`, "success");
          }
          break;
        }
        case "match.removed": {
          const data = event.data as { userId: string };
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          client.removeQueries({ queryKey: queryKeys.messages(data.userId) });
          break;
        }
        case "notification.created":
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        default:
          break;
      }
    };

    const realtime = new RealtimeClient(handle);
    realtime.start();
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        realtime.wake();
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
        void client.invalidateQueries({ queryKey: queryKeys.notifications });
      }
    });
    return () => {
      subscription.remove();
      realtime.stop();
    };
  }, [client, status, userId]);

  return null;
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  return (
    <>
      <RealtimeBridge />
      {children}
    </>
  );
}
