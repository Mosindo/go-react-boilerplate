import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import type { Message, MessagesPage } from "../api/types";

export const queryKeys = {
  me: ["profile", "me"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  profile: (userId: string) => ["profile", userId] as const
};

export type MessagesData = InfiniteData<MessagesPage, string | undefined>;

/** Appends a message to the newest page of a conversation's cache, ignoring duplicates (echo of our own send). */
export function appendMessage(data: MessagesData | undefined, message: Message): MessagesData | undefined {
  if (!data || data.pages.length === 0) return data;
  if (data.pages.some((page) => page.messages.some((m) => m.id === message.id))) return data;
  const [newest, ...older] = data.pages;
  return { ...data, pages: [{ ...newest, messages: [...newest.messages, message] }, ...older] };
}

export function applyMessageNew(client: QueryClient, message: Message): void {
  client.setQueryData<MessagesData>(queryKeys.messages(message.conversationId), (data) => appendMessage(data, message));
  void client.invalidateQueries({ queryKey: queryKeys.conversations });
  void client.invalidateQueries({ queryKey: queryKeys.matches });
}

export function applyMessagesRead(client: QueryClient, conversationId: string, readerUserId: string, readAt: string): void {
  // The reader read what the *other* person sent them, i.e. everything not sent by the reader.
  client.setQueryData<MessagesData>(queryKeys.messages(conversationId), (data) => {
    if (!data) return data;
    return {
      ...data,
      pages: data.pages.map((page) => ({
        ...page,
        messages: page.messages.map((m) => (m.senderUserId !== readerUserId && !m.readAt ? { ...m, readAt } : m))
      }))
    };
  });
  void client.invalidateQueries({ queryKey: queryKeys.conversations });
}

export function applyMatchRemoved(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: queryKeys.matches });
  void client.invalidateQueries({ queryKey: queryKeys.conversations });
  client.removeQueries({ queryKey: ["messages"] });
}
