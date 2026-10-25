import type { Message } from "../../api";
import { isLocalMessage } from "../../realtime/cache";
import { dayKey, formatDayLabel } from "../../shared/format";

export type MessageStatus = "sending" | "sent" | "read";

export type ThreadRow =
  | { kind: "message"; key: string; message: Message; mine: boolean; status: MessageStatus | null }
  | { kind: "day"; key: string; label: string };

/**
 * Turns a newest-first message list into rows for an inverted list. A day separator is emitted after the last
 * (oldest) message of each day, which renders above it. The status line is shown only under my latest message.
 */
export function buildThreadRows(
  messages: Message[],
  currentUserId: string,
  hasOlderMessages: boolean,
  now: Date = new Date()
): ThreadRow[] {
  const rows: ThreadRow[] = [];
  let statusAssigned = false;

  messages.forEach((message, index) => {
    const mine = message.senderId === currentUserId;
    let status: MessageStatus | null = null;
    if (mine && !statusAssigned) {
      statusAssigned = true;
      status = isLocalMessage(message) ? "sending" : message.readAt ? "read" : "sent";
    }
    rows.push({ kind: "message", key: message.id, message, mine, status });

    const older = messages[index + 1];
    const startsNewDay = older ? dayKey(older.createdAt) !== dayKey(message.createdAt) : !hasOlderMessages;
    if (startsNewDay) {
      rows.push({
        kind: "day",
        key: `day-${dayKey(message.createdAt)}`,
        label: formatDayLabel(message.createdAt, now)
      });
    }
  });

  return rows;
}
