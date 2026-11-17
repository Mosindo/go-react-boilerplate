import type { InfiniteData } from "@tanstack/react-query";
import type { Message, MessagesPage } from "../api/types";

/**
 * Inserts a message at the head of the newest page, replacing an optimistic copy with the
 * same clientMessageId and ignoring duplicates (the sender receives its own echo).
 */
export function insertMessage(data: InfiniteData<MessagesPage> | undefined, message: Message): InfiniteData<MessagesPage> | undefined {
  if (!data || data.pages.length === 0) return data;
  const exists = data.pages.some((page) => page.messages.some((m) => m.id === message.id));
  const pages = data.pages.map((page, index) => {
    let messages = page.messages;
    if (message.clientMessageId) {
      messages = messages.filter((m) => !(m.id.startsWith("local-") && m.clientMessageId === message.clientMessageId));
    }
    if (index === 0 && !exists) {
      messages = [message, ...messages];
    }
    return { ...page, messages };
  });
  return { ...data, pages };
}
