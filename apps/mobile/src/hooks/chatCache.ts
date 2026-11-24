import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import type { Message, MessagesPage } from "../api/types";
import { queryKeys } from "../queryClient";

/** Inserts a message at the top of the newest page, ignoring duplicates. */
export function insertMessage(client: QueryClient, message: Message) {
  client.setQueryData<InfiniteData<MessagesPage, string | undefined>>(queryKeys.messages(message.conversationId), (data) => {
    if (!data) return data;
    if (data.pages.some((page) => page.messages.some((m) => m.id === message.id))) return data;
    const [first, ...rest] = data.pages;
    return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
  });
}

/** Marks my messages as read by the other member up to lastReadAt. */
export function markOwnMessagesRead(client: QueryClient, conversationId: string, myUserId: string, lastReadAt: string) {
  const limit = new Date(lastReadAt).getTime();
  client.setQueryData<InfiniteData<MessagesPage, string | undefined>>(queryKeys.messages(conversationId), (data) => {
    if (!data) return data;
    return {
      ...data,
      pages: data.pages.map((page) => ({
        ...page,
        otherLastReadAt: lastReadAt,
        messages: page.messages.map((m) =>
          m.senderId === myUserId && !m.isRead && new Date(m.createdAt).getTime() <= limit ? { ...m, isRead: true } : m
        )
      }))
    };
  });
}
