import type { Message } from "../api/types";

export type PendingMessage = { tempId: string; content: string; createdAt: string; failed: boolean };

const ECHO_WINDOW_MS = 10_000;

/**
 * Drops optimistic messages that the server already confirmed. The realtime echo of our own message can
 * arrive before the HTTP response, so for a moment both exist; each pending message is matched with at most
 * one confirmed message of ours with the same text sent after it, so two identical quick messages stay two.
 */
export function visiblePending(messages: Message[], pending: PendingMessage[], myId: string): PendingMessage[] {
  const used = new Set<string>();
  return pending.filter((p) => {
    if (p.failed) return true;
    const sentAt = Date.parse(p.createdAt);
    const confirmed = messages.find(
      (m) => !used.has(m.id) && m.senderUserId === myId && m.content === p.content && Date.parse(m.createdAt) >= sentAt - ECHO_WINDOW_MS
    );
    if (!confirmed) return true;
    used.add(confirmed.id);
    return false;
  });
}
