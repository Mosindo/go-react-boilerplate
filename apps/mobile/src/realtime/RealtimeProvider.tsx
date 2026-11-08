import React, { useEffect } from "react";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import { keys } from "../hooks/queries";
import { wasCelebrated } from "../features/discover/matchCelebration";
import { mergeMessage } from "../lib/messages";
import { useToast } from "../ui/Toast";
import { RealtimeClient } from "./client";
import type { MessagesPage, RealtimeEvent } from "../api/types";

/** Keeps the React Query cache in sync with server events pushed over the WebSocket. */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const queryClient = useQueryClient();
  const toast = useToast();

  useEffect(() => {
    if (status !== "signedIn") return;
    const client = new RealtimeClient();
    let wasConnected = false;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "ready":
          // After a (re)connection, refetch so nothing missed while offline is lost.
          if (wasConnected) void queryClient.invalidateQueries();
          wasConnected = true;
          break;
        case "message.new": {
          const message = event.data;
          queryClient.setQueryData<InfiniteData<MessagesPage, string | undefined>>(
            keys.messages(message.conversationId),
            (data) => {
              if (!data || data.pages.length === 0) return data;
              const [first, ...rest] = data.pages;
              return {
                ...data,
                pages: [{ ...first, messages: mergeMessage(first.messages, message) }, ...rest]
              };
            }
          );
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.unread });
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          break;
        }
        case "match.new":
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          {
            // The person who triggered the match already gets the full-screen celebration from the swipe
            // response; only the other side needs this toast. The short delay lets that response arrive first.
            const { id, user } = event.data;
            setTimeout(() => {
              if (!wasCelebrated(id)) toast.show(`C'est un match avec ${user.firstName} !`, "success");
            }, 1000);
          }
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "conversation.read":
          queryClient.setQueryData<InfiniteData<MessagesPage, string | undefined>>(
            keys.messages(event.data.conversationId),
            (data) =>
              data
                ? { ...data, pages: data.pages.map((p) => ({ ...p, peerLastReadAt: event.data.readAt })) }
                : data
          );
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
      }
    };

    const off = client.on(handle);
    client.start();
    return () => {
      off();
      client.stop();
    };
  }, [status, queryClient, toast]);

  return <>{children}</>;
}
