import type { Message } from "../api/types";

export type ThreadRow =
  | { kind: "day"; key: string; label: string; iso: string }
  | { kind: "message"; key: string; message: Message; mine: boolean; showReceipt: boolean };

const dayKey = (iso: string) => {
  const d = new Date(iso);
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
};

/**
 * Builds the rows of a chat in newest-first order (what an inverted FlatList wants),
 * inserting a day separator before the first message of each day and a "read" receipt
 * under the user's latest message when the other person has read it.
 */
export function buildThreadRows(messages: readonly Message[], currentUserId: string, dayLabel: (iso: string) => string): ThreadRow[] {
  const chronological: ThreadRow[] = [];
  let previousDay = "";
  let lastMineIndex = -1;

  messages.forEach((message) => {
    const day = dayKey(message.createdAt);
    if (day !== previousDay) {
      chronological.push({ kind: "day", key: `day-${day}`, label: dayLabel(message.createdAt), iso: message.createdAt });
      previousDay = day;
    }
    const mine = message.senderId === currentUserId;
    if (mine) {
      lastMineIndex = chronological.length;
    }
    chronological.push({ kind: "message", key: message.id, message, mine, showReceipt: false });
  });

  const last = chronological[lastMineIndex];
  if (last && last.kind === "message" && last.message.readAt) {
    chronological[lastMineIndex] = { ...last, showReceipt: true };
  }
  return chronological.reverse();
}

/** True when the newest incoming message has not been read yet. */
export function hasUnreadIncoming(messages: readonly Message[], currentUserId: string): boolean {
  return messages.some((m) => m.senderId !== currentUserId && m.readAt === null);
}

/** Merges an older page in front of the cache, dropping messages already known. */
export function prependPage(existing: readonly Message[], older: readonly Message[]): Message[] {
  const known = new Set(existing.map((m) => m.id));
  return [...older.filter((m) => !known.has(m.id)), ...existing];
}
