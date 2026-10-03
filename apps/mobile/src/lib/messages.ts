import type { ChatMessage } from "../api/types";

/** Inserts a message into a newest-first list, ignoring duplicates (REST reply + push). */
export function mergeMessage(list: ChatMessage[], incoming: ChatMessage): ChatMessage[] {
  if (list.some((m) => m.id === incoming.id)) {
    return list;
  }
  const next = [incoming, ...list];
  next.sort((a, b) => (a.createdAt < b.createdAt ? 1 : a.createdAt > b.createdAt ? -1 : 0));
  return next;
}

/** Marks everything the reader's counterpart sent as read. */
export function applyReadReceipt(list: ChatMessage[], readerId: string, readAt: string): ChatMessage[] {
  return list.map((m) => (m.senderId !== readerId && !m.readAt ? { ...m, readAt } : m));
}

/** Appends an older page, dropping anything already present. */
export function appendOlder(list: ChatMessage[], older: ChatMessage[]): ChatMessage[] {
  const known = new Set(list.map((m) => m.id));
  return [...list, ...older.filter((m) => !known.has(m.id))];
}
