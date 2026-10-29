import type { InfiniteData } from "@tanstack/react-query";
import type { Message, MessagesPage } from "../api/types";

export type MessagesCache = InfiniteData<MessagesPage, unknown>;

/**
 * Pages are fetched newest first: pages[0] holds the latest messages (oldest to
 * newest inside the page), later pages hold older history.
 */
export function appendMessage(
  cache: MessagesCache | undefined,
  message: Message
): MessagesCache | undefined {
  if (!cache || cache.pages.length === 0) {
    return cache;
  }
  const exists = cache.pages.some((page) => page.messages.some((m) => m.id === message.id));
  if (exists) {
    return cache;
  }
  const [first, ...rest] = cache.pages;
  if (!first) {
    return cache;
  }
  return { ...cache, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
}

/** Marks every message sent by someone other than `readerId` as read. */
export function markReadBy(
  cache: MessagesCache | undefined,
  readerId: string,
  readAt: string
): MessagesCache | undefined {
  if (!cache) {
    return cache;
  }
  return {
    ...cache,
    pages: cache.pages.map((page) => ({
      ...page,
      messages: page.messages.map((m) =>
        m.senderId !== readerId && m.readAt === null ? { ...m, readAt } : m
      )
    }))
  };
}

/** Flattens cached pages into one oldest-to-newest list without duplicates. */
export function flattenMessages(cache: MessagesCache | undefined): Message[] {
  if (!cache) {
    return [];
  }
  const seen = new Set<number>();
  const out: Message[] = [];
  for (let i = cache.pages.length - 1; i >= 0; i -= 1) {
    for (const message of cache.pages[i]?.messages ?? []) {
      if (!seen.has(message.id)) {
        seen.add(message.id);
        out.push(message);
      }
    }
  }
  return out;
}
