import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import type { Message } from "../api/types";
import { keys } from "./keys";

type Pages = InfiniteData<Message[], string | undefined>;

/** Inserts a message at the front (newest first) of a cached conversation, ignoring duplicates. */
export function insertMessage(data: Pages | undefined, message: Message): Pages | undefined {
  if (!data || data.pages.length === 0) {
    return data;
  }
  if (data.pages.some((page) => page.some((m) => m.id === message.id))) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [[message, ...first], ...rest] };
}

export function prependMessage(queryClient: QueryClient, message: Message): void {
  queryClient.setQueryData<Pages>(keys.messages(message.conversationId), (data) => insertMessage(data, message));
}
