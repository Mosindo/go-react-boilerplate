import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import type { Message } from "../api/types";

export const queryKeys = {
  conversations: ["conversations"] as const,
  chatUnread: ["chat-unread"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  notificationsUnread: ["notifications-unread"] as const,
  ownProfile: ["own-profile"] as const,
  interests: ["interests"] as const,
  card: (userId: string) => ["card", userId] as const,
  blocked: ["blocked"] as const,
};

export type MessagePages = InfiniteData<Message[], string | undefined>;

/** Adds a message to the cached first page (newest first) unless it is already there. */
export function insertMessage(
  data: MessagePages | undefined,
  message: Message,
): MessagePages | undefined {
  if (!data) return data;
  const exists = data.pages.some((page) => page.some((m) => m.id === message.id));
  if (exists) return data;
  const [first = [], ...rest] = data.pages;
  return { ...data, pages: [[message, ...first], ...rest] };
}

/** Marks every message not sent by `readerId` as read, so read receipts update live. */
export function markMessagesRead(
  data: MessagePages | undefined,
  readerId: string,
  readAt: string,
): MessagePages | undefined {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((page) =>
      page.map((m) => (m.senderId !== readerId && !m.readAt ? { ...m, readAt } : m)),
    ),
  };
}

export function invalidateChat(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
  void queryClient.invalidateQueries({ queryKey: queryKeys.chatUnread });
}
