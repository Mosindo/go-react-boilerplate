import type { InfiniteData } from "@tanstack/react-query";
import type { Message } from "../api/types";

export type MessagePages = InfiniteData<
  { messages: Message[]; hasMore: boolean },
  number | undefined
>;

/** Inserts a message at the head of the newest page (newest-first), ignoring duplicates. */
export function insertMessage(
  data: MessagePages | undefined,
  msg: Message
): MessagePages | undefined {
  if (!data || data.pages.length === 0) return data;
  if (data.pages.some((p) => p.messages.some((m) => m.id === msg.id))) return data;
  const [first, ...rest] = data.pages;
  if (!first) return data;
  return { ...data, pages: [{ ...first, messages: [msg, ...first.messages] }, ...rest] };
}

/** Flattens pages (newest-first) into one list. */
export function flattenMessages(data: MessagePages | undefined): Message[] {
  return data ? data.pages.flatMap((p) => p.messages) : [];
}
