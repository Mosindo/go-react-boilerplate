import type { Message } from "../../api/models";
import { dayKey, formatDayLabel } from "./format";

export const MAX_MESSAGE_LENGTH = 2000;
export const COUNTER_VISIBLE_FROM = 1800;

export type MessagesPage = { messages: Message[]; nextCursor: string | null };

export type PendingStatus = "sending" | "failed";

/** A message the user typed that the server has not acknowledged yet. */
export type PendingMessage = {
  localId: string;
  body: string;
  createdAt: string;
  status: PendingStatus;
  error?: string;
};

export type ThreadMessageItem = {
  kind: "message";
  key: string;
  message: Message;
  mine: boolean;
  pending: PendingStatus | null;
  localId: string | null;
  /** True when the next-older item is a message from the same sender (tighter spacing). */
  groupedWithOlder: boolean;
};

export type ThreadSeparatorItem = { kind: "separator"; key: string; label: string };

export type ThreadItem = ThreadMessageItem | ThreadSeparatorItem;

function compareNewestFirst(a: Message, b: Message): number {
  if (a.createdAt !== b.createdAt) return a.createdAt < b.createdAt ? 1 : -1;
  if (a.id === b.id) return 0;
  return a.id < b.id ? 1 : -1;
}

/** Flatten cached pages (newest first) into one list, deduplicated by id, newest first. */
export function flattenMessages(pages: readonly MessagesPage[]): Message[] {
  const seen = new Set<string>();
  const out: Message[] = [];
  for (const page of pages) {
    for (const m of page.messages) {
      if (seen.has(m.id)) continue;
      seen.add(m.id);
      out.push(m);
    }
  }
  return out.sort(compareNewestFirst);
}

/** Insert or update a message in the cached pages. Idempotent. */
export function upsertMessage<D extends { pages: MessagesPage[] }>(data: D, message: Message): D {
  let exists = false;
  const pages = data.pages.map((page) => {
    if (!page.messages.some((m) => m.id === message.id)) return page;
    exists = true;
    return {
      ...page,
      messages: page.messages.map((m) => (m.id === message.id ? { ...m, ...message } : m))
    };
  });
  if (exists) return { ...data, pages };
  if (pages.length === 0) {
    return { ...data, pages: [{ messages: [message], nextCursor: null }] };
  }
  const [first, ...rest] = pages;
  const messages = [message, ...first.messages].sort(compareNewestFirst);
  return { ...data, pages: [{ ...first, messages }, ...rest] };
}

/** Mark every message I sent as read (the other participant opened the conversation). */
export function markMineRead<D extends { pages: MessagesPage[] }>(
  data: D,
  myUserId: string,
  readAt: string
): D {
  let changed = false;
  const pages = data.pages.map((page) => {
    let pageChanged = false;
    const messages = page.messages.map((m) => {
      if (m.senderId === myUserId && m.readAt === null) {
        pageChanged = true;
        return { ...m, readAt };
      }
      return m;
    });
    if (!pageChanged) return page;
    changed = true;
    return { ...page, messages };
  });
  return changed ? { ...data, pages } : data;
}

/**
 * Build the render list for an inverted FlatList (index 0 = newest, bottom of the screen).
 * Pending (optimistic) messages are always shown as the newest; the caller removes a pending entry as soon
 * as the POST response is merged into the cache with `upsertMessage`, which is idempotent by id, so a
 * realtime echo that beats the response never produces a duplicate server message.
 * A date separator is inserted after the oldest message of each day (i.e. displayed above it).
 */
export function buildThreadItems(
  serverMessages: readonly Message[],
  pending: readonly PendingMessage[],
  myUserId: string,
  now: Date = new Date()
): ThreadItem[] {
  const pendingMessages: { message: Message; pending: PendingMessage }[] = [...pending]
    .sort((a, b) => (a.createdAt < b.createdAt ? 1 : a.createdAt > b.createdAt ? -1 : 0))
    .map((p) => ({
      pending: p,
      message: {
        id: p.localId,
        conversationId: "",
        senderId: myUserId,
        body: p.body,
        createdAt: p.createdAt,
        readAt: null
      }
    }));

  const rows: { message: Message; pending: PendingMessage | null }[] = [
    ...pendingMessages,
    ...serverMessages.map((message) => ({ message, pending: null }))
  ];

  const items: ThreadItem[] = [];
  for (let i = 0; i < rows.length; i += 1) {
    const { message, pending: pend } = rows[i];
    const older = rows[i + 1];
    items.push({
      kind: "message",
      key: message.id,
      message,
      mine: message.senderId === myUserId,
      pending: pend ? pend.status : null,
      localId: pend ? pend.localId : null,
      groupedWithOlder:
        !!older &&
        older.message.senderId === message.senderId &&
        dayKey(older.message.createdAt) === dayKey(message.createdAt)
    });
    const isOldestOfDay = !older || dayKey(older.message.createdAt) !== dayKey(message.createdAt);
    if (isOldestOfDay) {
      items.push({
        kind: "separator",
        key: `sep-${dayKey(message.createdAt)}`,
        label: formatDayLabel(message.createdAt, now)
      });
    }
  }
  return items;
}

export function canSend(draft: string): boolean {
  const trimmed = draft.trim();
  return trimmed.length > 0 && trimmed.length <= MAX_MESSAGE_LENGTH;
}

export function showCounter(length: number): boolean {
  return length >= COUNTER_VISIBLE_FROM;
}

export type SendFailure = "forbidden" | "gone" | "retry";

/** Map an HTTP status (null for network failures) to how the thread must react. */
export function classifySendFailure(status: number | null): SendFailure {
  if (status === 403) return "forbidden";
  if (status === 404) return "gone";
  return "retry";
}
