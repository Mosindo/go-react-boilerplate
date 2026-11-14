import React, { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import type { AppNotification, Conversation, Match, Message, MessagesPage } from "../api/types";
import { keys } from "../state/queryClient";
import { bumpConversation, insertMessage, setOtherReadAt } from "../state/queries";
import { useAuth } from "../state/AuthProvider";
import { useToast } from "../ui";
import { navigationRef } from "../navigation/navigationRef";
import { realtime } from "./socket";

// The conversation currently on screen (no toast, auto mark-as-read).
let activeMatchId: string | null = null;
export function setActiveConversation(matchId: string | null) {
  activeMatchId = matchId;
}
export function getActiveConversation() {
  return activeMatchId;
}

type MessagesCache = InfiniteData<MessagesPage, string | undefined>;

/** Bridges server push events into the React Query cache. */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status, user } = useAuth();
  const qc = useQueryClient();
  const toast = useToast();

  useEffect(() => {
    if (status !== "signedIn" || !user) {
      return;
    }
    realtime.start();
    const unsubscribe = realtime.subscribe((event) => {
      switch (event.type) {
        case "ready":
          // (Re)connected: resync anything we may have missed.
          void qc.invalidateQueries({ queryKey: keys.conversations });
          void qc.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "message.created": {
          const message = event.data as Message;
          const mine = message.senderId === user.id;
          const isActive = activeMatchId === message.matchId;
          qc.setQueryData<MessagesCache>(keys.messages(message.matchId), (cache) => insertMessage(cache, message));
          const list = qc.getQueryData<Conversation[]>(keys.conversations);
          if (list?.some((c) => c.matchId === message.matchId)) {
            qc.setQueryData<Conversation[]>(keys.conversations, (l) => bumpConversation(l, message, mine || isActive ? 0 : 1));
          } else {
            void qc.invalidateQueries({ queryKey: keys.conversations });
          }
          break;
        }
        case "conversation.read": {
          const { matchId, readAt } = event.data as { matchId: string; readAt: string };
          qc.setQueryData<MessagesCache>(keys.messages(matchId), (cache) => setOtherReadAt(cache, readAt));
          qc.setQueryData<Conversation[]>(keys.conversations, (l) =>
            l?.map((c) => (c.matchId === matchId ? { ...c, otherLastReadAt: readAt } : c))
          );
          break;
        }
        case "match.created": {
          const match = event.data as Match;
          void qc.invalidateQueries({ queryKey: keys.conversations });
          toast({
            title: `C'est un match avec ${match.user.firstName} !`,
            body: "Envoyez-lui un premier message.",
            onPress: () => navigationRef.isReady() && navigationRef.navigate("Conversation", { matchId: match.id, user: match.user })
          });
          break;
        }
        case "match.removed": {
          const { matchId } = event.data as { matchId: string };
          qc.setQueryData<Conversation[]>(keys.conversations, (l) => l?.filter((c) => c.matchId !== matchId));
          qc.removeQueries({ queryKey: keys.messages(matchId) });
          break;
        }
        case "notification.created": {
          const n = event.data as AppNotification;
          void qc.invalidateQueries({ queryKey: keys.notifications });
          if (n.type === "message" && n.data?.matchId && n.data.matchId !== activeMatchId) {
            toast({
              title: n.title,
              body: n.body,
              onPress: () => {
                const conv = qc.getQueryData<Conversation[]>(keys.conversations)?.find((c) => c.matchId === n.data?.matchId);
                if (conv && navigationRef.isReady()) {
                  navigationRef.navigate("Conversation", { matchId: conv.matchId, user: conv.user });
                }
              }
            });
          }
          break;
        }
        default:
          break;
      }
    });
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        realtime.reconnect();
      }
    });
    return () => {
      unsubscribe();
      appState.remove();
      realtime.stop();
    };
  }, [qc, status, toast, user]);

  return <>{children}</>;
}
