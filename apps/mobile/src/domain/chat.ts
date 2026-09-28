import type { ChatMessage } from "../api/types";

const GROUP_WINDOW_MS = 5 * 60 * 1000;

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

export function formatClock(date: Date): string {
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}`;
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const dayDiff = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (dayDiff === 0) {
    return "Today";
  }
  if (dayDiff === 1) {
    return "Yesterday";
  }
  if (dayDiff > 1 && dayDiff < 7) {
    return WEEKDAYS[date.getDay()];
  }
  const base = `${date.getDate()} ${MONTHS[date.getMonth()]}`;
  return date.getFullYear() === now.getFullYear() ? base : `${base} ${date.getFullYear()}`;
}

/** Short label for conversation lists: clock time today, otherwise the day. */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  return startOfDay(now) === startOfDay(date) ? formatClock(date) : formatDayLabel(iso, now);
}

export type ChatRow =
  | { kind: "message"; key: string; message: ChatMessage; showTime: boolean; startsGroup: boolean }
  | { kind: "day"; key: string; label: string };

/**
 * Turns messages (newest first, as served by the API) into rows for an inverted list:
 * day separators, and a timestamp only on the last message of a same-sender burst.
 */
export function buildChatRows(messagesNewestFirst: ChatMessage[], now: Date = new Date()): ChatRow[] {
  const rows: ChatRow[] = [];
  messagesNewestFirst.forEach((message, index) => {
    const older = messagesNewestFirst[index + 1];
    const newer = messagesNewestFirst[index - 1];
    const created = new Date(message.createdAt).getTime();
    const sameGroupAsNewer =
      newer !== undefined &&
      newer.senderId === message.senderId &&
      new Date(newer.createdAt).getTime() - created < GROUP_WINDOW_MS &&
      startOfDay(new Date(newer.createdAt)) === startOfDay(new Date(message.createdAt));
    const sameGroupAsOlder =
      older !== undefined &&
      older.senderId === message.senderId &&
      created - new Date(older.createdAt).getTime() < GROUP_WINDOW_MS &&
      startOfDay(new Date(older.createdAt)) === startOfDay(new Date(message.createdAt));
    rows.push({
      kind: "message",
      key: message.id,
      message,
      showTime: !sameGroupAsNewer,
      startsGroup: !sameGroupAsOlder
    });
    const startsDay = !older || startOfDay(new Date(older.createdAt)) !== startOfDay(new Date(message.createdAt));
    if (startsDay) {
      rows.push({ kind: "day", key: `day-${message.id}`, label: formatDayLabel(message.createdAt, now) });
    }
  });
  return rows;
}

/** Text-only delivery glyphs: single check = delivered, double check = read. */
export function receiptGlyph(message: ChatMessage): string {
  if (message.status === "sending") {
    return "…";
  }
  if (message.status === "failed") {
    return "!";
  }
  return message.readAt ? "✓✓" : "✓";
}

export function receiptLabel(message: ChatMessage): string {
  if (message.status === "sending") {
    return "Sending";
  }
  if (message.status === "failed") {
    return "Not sent. Tap to retry";
  }
  return message.readAt ? "Read" : "Delivered";
}
