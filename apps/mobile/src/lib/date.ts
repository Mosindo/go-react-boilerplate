import { fr } from '../i18n';
import type { Message } from '../api/types';

const pad = (n: number): string => String(n).padStart(2, '0');

/** Local calendar day key, e.g. 2026-03-05. */
export function dayKey(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

export function formatTime(iso: string): string {
  const date = new Date(iso);
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

const dayFormatter = new Intl.DateTimeFormat('fr-FR', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
});
const dayYearFormatter = new Intl.DateTimeFormat('fr-FR', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
});

export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const today = dayKey(now);
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
  const key = dayKey(date);
  if (key === today) return fr.dates.today;
  if (key === dayKey(yesterday)) return fr.dates.yesterday;
  return date.getFullYear() === now.getFullYear()
    ? dayFormatter.format(date)
    : dayYearFormatter.format(date);
}

/** Short relative label for conversation rows: HH:MM today, "Hier", weekday, then dd/mm. */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffDays = Math.floor(
    (new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime() -
      new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()) /
      86_400_000,
  );
  if (diffDays <= 0) return formatTime(iso);
  if (diffDays === 1) return fr.dates.yesterday;
  if (diffDays < 7) {
    const label = new Intl.DateTimeFormat('fr-FR', { weekday: 'short' }).format(date);
    return label.replace('.', '');
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

export type ChatListItem =
  { kind: 'message'; key: string; message: Message } | { kind: 'day'; key: string; label: string };

/**
 * Input: messages newest first (inverted list order).
 * Output: same order with a day separator placed after (= visually above) the oldest message of each day.
 */
export function buildChatItems(messages: Message[], now: Date = new Date()): ChatListItem[] {
  const items: ChatListItem[] = [];
  messages.forEach((message, index) => {
    items.push({ kind: 'message', key: message.id, message });
    const older = messages[index + 1];
    const currentDay = dayKey(new Date(message.createdAt));
    if (!older || dayKey(new Date(older.createdAt)) !== currentDay) {
      items.push({
        kind: 'day',
        key: `day-${currentDay}`,
        label: formatDayLabel(message.createdAt, now),
      });
    }
  });
  return items;
}
