import type { InfiniteData } from "@tanstack/react-query";
import type { Message } from "../api";

export type MessagePage = { messages: Message[]; hasMore: boolean };

/** Adds a message (newest first) to the first page unless it is already cached. */
export function insertMessage(data: InfiniteData<MessagePage, number | undefined> | undefined, msg: Message) {
  if (!data || data.pages.length === 0) return data;
  if (data.pages.some((p) => p.messages.some((m) => m.id === msg.id))) return data;
  const [first, ...rest] = data.pages;
  if (!first) return data;
  return { ...data, pages: [{ ...first, messages: [msg, ...first.messages] }, ...rest] };
}

/** Marks every message sent by someone other than the reader as read. */
export function applyReadReceipt(
  data: InfiniteData<MessagePage, number | undefined> | undefined,
  readerId: string,
  readAt: string
) {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      messages: p.messages.map((m) => (m.senderId !== readerId && !m.readAt ? { ...m, readAt } : m))
    }))
  };
}
