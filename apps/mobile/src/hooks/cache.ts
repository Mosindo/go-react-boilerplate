import type { InfiniteData } from "@tanstack/react-query";
import type { Message, MessagesPage } from "../api/types";

/** Inserts a message at the top of a cached, newest-first message list. */
export function insertMessage(data: InfiniteData<MessagesPage, string | null> | undefined, message: Message) {
  if (!data || data.pages.length === 0) {
    return data;
  }
  if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}
