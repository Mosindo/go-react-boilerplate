import type { Message } from "../types";

export const MAX_MESSAGE_LENGTH = 2000;

export type PendingMessage = {
  localId: string;
  conversationId: string;
  senderId: string;
  content: string;
  createdAt: string;
  status: "sending" | "failed";
};

export type ChatMessageView = {
  /** Stable list key: server id, or localId for unsent messages. */
  key: string;
  senderId: string;
  content: string;
  createdAt: string;
  status: "sent" | "sending" | "failed";
  localId?: string;
};

export type ChatListItem =
  | { kind: "message"; key: string; message: ChatMessageView; showSeen: boolean; showTime: boolean }
  | { kind: "separator"; key: string; label: string };

function ts(iso: string): number {
  const value = Date.parse(iso);
  return Number.isNaN(value) ? 0 : value;
}

function byNewest(a: Message, b: Message): number {
  const diff = ts(b.createdAt) - ts(a.createdAt);
  if (diff !== 0) {
    return diff;
  }
  return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
}

/** Flatten cursor pages (each newest first) into one deduplicated newest-first list. */
export function flattenPages(pages: ReadonlyArray<{ items: Message[] }>): Message[] {
  const seen = new Set<string>();
  const out: Message[] = [];
  for (const page of pages) {
    for (const message of page.items) {
      if (!seen.has(message.id)) {
        seen.add(message.id);
        out.push(message);
      }
    }
  }
  return out.sort(byNewest);
}

/** Insert or replace a message in a newest-first list. Returns the same array when nothing changed. */
export function upsertMessage(list: Message[], message: Message): Message[] {
  const index = list.findIndex((m) => m.id === message.id);
  if (index >= 0) {
    const current = list[index];
    if (current.content === message.content && current.createdAt === message.createdAt) {
      return list;
    }
    const copy = list.slice();
    copy[index] = message;
    return copy.sort(byNewest);
  }
  return [...list, message].sort(byNewest);
}

const RECONCILE_WINDOW_MS = 60_000;

/**
 * Combine server messages with local optimistic ones (newest first).
 * A "sending" message is considered delivered as soon as an unclaimed server message from the same
 * sender with identical content appears (realtime echo can beat the HTTP response).
 */
export function mergeWithPending(server: Message[], pending: PendingMessage[]): ChatMessageView[] {
  const claimed = new Set<string>();
  const unsent: ChatMessageView[] = [];

  for (const p of pending) {
    if (p.status === "sending") {
      const match = server.find(
        (m) =>
          !claimed.has(m.id) &&
          m.senderId === p.senderId &&
          m.content === p.content &&
          ts(m.createdAt) >= ts(p.createdAt) - RECONCILE_WINDOW_MS
      );
      if (match) {
        claimed.add(match.id);
        continue;
      }
    }
    unsent.push({
      key: p.localId,
      localId: p.localId,
      senderId: p.senderId,
      content: p.content,
      createdAt: p.createdAt,
      status: p.status
    });
  }

  unsent.sort((a, b) => ts(b.createdAt) - ts(a.createdAt));

  const sent: ChatMessageView[] = server.map((m) => ({
    key: m.id,
    senderId: m.senderId,
    content: m.content,
    createdAt: m.createdAt,
    status: "sent"
  }));

  return [...unsent, ...sent];
}

/** Key of my most recent message that the peer has read, or null. */
export function lastSeenMessageKey(
  views: ChatMessageView[],
  myId: string,
  peerLastReadAt: string | null
): string | null {
  if (!peerLastReadAt) {
    return null;
  }
  const readAt = ts(peerLastReadAt);
  for (const view of views) {
    if (view.status === "sent" && view.senderId === myId && ts(view.createdAt) <= readAt) {
      return view.key;
    }
  }
  return null;
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

function dayKey(date: Date): string {
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export function dayLabel(iso: string, now: Date): string {
  const date = new Date(iso);
  const diffDays = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays === 0) {
    return "Today";
  }
  if (diffDays === 1) {
    return "Yesterday";
  }
  const base = `${WEEKDAYS[date.getDay()]}, ${MONTHS[date.getMonth()]} ${date.getDate()}`;
  return date.getFullYear() === now.getFullYear() ? base : `${base}, ${date.getFullYear()}`;
}

export function formatClock(iso: string): string {
  const date = new Date(iso);
  const hours = date.getHours();
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const suffix = hours >= 12 ? "PM" : "AM";
  const h12 = hours % 12 === 0 ? 12 : hours % 12;
  return `${h12}:${minutes} ${suffix}`;
}

/** Compact timestamp for the conversation list. */
export function formatListTime(iso: string, now: Date): string {
  const date = new Date(iso);
  const diffDays = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays <= 0) {
    return formatClock(iso);
  }
  if (diffDays === 1) {
    return "Yesterday";
  }
  if (diffDays < 7) {
    return WEEKDAYS[date.getDay()];
  }
  const base = `${MONTHS[date.getMonth()]} ${date.getDate()}`;
  return date.getFullYear() === now.getFullYear() ? base : `${base}, ${date.getFullYear()}`;
}

/**
 * Build the render list for an inverted FlatList (index 0 = newest, drawn at the bottom).
 * A date separator is placed after (above, visually) the oldest message of each day.
 * Only the newest message of a run within the same minute and sender shows its time.
 */
export function buildChatItems(
  views: ChatMessageView[],
  myId: string,
  peerLastReadAt: string | null,
  now: Date
): ChatListItem[] {
  const seenKey = lastSeenMessageKey(views, myId, peerLastReadAt);
  const items: ChatListItem[] = [];
  for (let i = 0; i < views.length; i += 1) {
    const view = views[i];
    const older = views[i + 1];
    const newer = views[i - 1];
    const sameRunAsNewer =
      newer !== undefined &&
      newer.senderId === view.senderId &&
      Math.abs(ts(newer.createdAt) - ts(view.createdAt)) < 60_000;
    items.push({
      kind: "message",
      key: view.key,
      message: view,
      showSeen: seenKey === view.key,
      showTime: !sameRunAsNewer
    });
    const thisDay = dayKey(new Date(view.createdAt));
    if (older === undefined || dayKey(new Date(older.createdAt)) !== thisDay) {
      items.push({ kind: "separator", key: `sep-${thisDay}`, label: dayLabel(view.createdAt, now) });
    }
  }
  return items;
}

export function validateDraft(draft: string): { ok: boolean; content: string } {
  const content = draft.trim();
  return { ok: content.length > 0 && content.length <= MAX_MESSAGE_LENGTH, content };
}
