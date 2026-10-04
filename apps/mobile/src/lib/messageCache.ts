import type { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import type { ChatMessage, Message } from "../api/types";
import { localId } from "./format";
import { mapPages, prependItem, type CursorData } from "./pagination";

type MessagesData = CursorData<"messages", ChatMessage>;

/** Adds an optimistic message (status "sending") at the top of the conversation cache. */
export function addLocalMessage(
  client: QueryClient,
  conversationId: string,
  senderId: string,
  body: string,
  now: Date = new Date()
): ChatMessage {
  const local: ChatMessage = {
    id: localId("local"),
    conversationId,
    senderId,
    body,
    createdAt: now.toISOString(),
    readAt: null,
    status: "sending"
  };
  client.setQueryData<MessagesData>(queryKeys.messages(conversationId), (data) => prependItem(data, "messages", local));
  return local;
}

export function setLocalStatus(
  client: QueryClient,
  conversationId: string,
  localMessageId: string,
  status: "sending" | "failed"
): void {
  client.setQueryData<MessagesData>(queryKeys.messages(conversationId), (data) =>
    data
      ? mapPages(data, "messages", (items) => items.map((m) => (m.id === localMessageId ? { ...m, status } : m)))
      : data
  );
}

export function removeMessage(client: QueryClient, conversationId: string, messageId: string): void {
  client.setQueryData<MessagesData>(queryKeys.messages(conversationId), (data) =>
    data ? mapPages(data, "messages", (items) => items.filter((m) => m.id !== messageId)) : data
  );
}

/**
 * Swaps the optimistic entry for the server message. If the realtime event already inserted
 * the real message, the local copy is simply dropped (no duplicate).
 */
export function resolveLocalMessage(client: QueryClient, localMessageId: string, saved: Message): void {
  client.setQueryData<MessagesData>(queryKeys.messages(saved.conversationId), (data) => {
    if (!data) return data;
    const alreadyThere = data.pages.some((page) => page.messages.some((m) => m.id === saved.id));
    return mapPages(data, "messages", (items) =>
      alreadyThere
        ? items.filter((m) => m.id !== localMessageId)
        : items.map((m) => (m.id === localMessageId ? saved : m))
    );
  });
}
