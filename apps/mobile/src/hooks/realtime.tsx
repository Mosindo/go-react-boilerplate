import { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { RealtimeClient } from "../api/realtime";
import { tokenStore } from "../api/tokenStore";
import type { Message, MessagesPage, RealtimeEvent } from "../api/types";
import { useToast } from "../shared/ui";
import { useSession } from "./session";
import { keys } from "./queries";

type MessagesCache = InfiniteData<MessagesPage, string | undefined>;

/** Appends a message to the newest page unless it is already there. */
export function addMessageToCache(qc: QueryClient, message: Message): void {
  qc.setQueryData<MessagesCache>(keys.messages(message.conversationId), (old) => {
    if (!old || old.pages.length === 0) return old;
    const [first, ...rest] = old.pages;
    if (!first || first.messages.some((m) => m.id === message.id)) return old;
    return { ...old, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
  });
}

function handleEvent(qc: QueryClient, event: RealtimeEvent, myId: string, notify: (msg: string) => void): void {
  switch (event.type) {
    case "ready":
      // Catch up on anything missed while disconnected.
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.notifications });
      void qc.invalidateQueries({ queryKey: ["messages"] });
      break;
    case "message.new":
      addMessageToCache(qc, event.data);
      void qc.invalidateQueries({ queryKey: keys.conversations });
      break;
    case "conversation.read":
      if (event.data.readerId !== myId) {
        qc.setQueryData<MessagesCache>(keys.messages(event.data.conversationId), (old) =>
          old ? { ...old, pages: old.pages.map((p, i) => (i === 0 ? { ...p, peerLastReadId: event.data.lastReadId } : p)) } : old,
        );
      }
      break;
    case "conversation.removed":
      qc.removeQueries({ queryKey: keys.messages(event.data.conversationId) });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      break;
    case "match.new":
      notify("C'est un match !");
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.likes });
      break;
    case "notification.new":
      void qc.invalidateQueries({ queryKey: keys.notifications });
      break;
    default:
      break;
  }
}

/** Keeps one authenticated WebSocket open while signed in and feeds the query cache. */
export function RealtimeBridge() {
  const qc = useQueryClient();
  const { user } = useSession();
  const toast = useToast();
  const toastRef = useRef(toast);
  toastRef.current = toast;
  const myId = user?.id;

  useEffect(() => {
    if (!myId) return;
    const client = new RealtimeClient(async () => {
      try {
        await authApi.me(); // triggers a refresh if the access token expired
      } catch {
        // the stored token is used as-is; the server will refuse it if invalid
      }
      return (await tokenStore.get())?.accessToken ?? null;
    });
    const unsubscribe = client.subscribe((e) => handleEvent(qc, e, myId, (m) => toastRef.current.show(m, "success")));
    client.start();
    const appState = AppState.addEventListener("change", (s) => {
      if (s === "active") client.reconnectNow();
    });
    return () => {
      appState.remove();
      unsubscribe();
      client.stop();
    };
  }, [qc, myId]);

  return null;
}
