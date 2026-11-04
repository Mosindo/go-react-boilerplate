export type WithId = { id: string };

/**
 * Pages are fetched newest-first, each page being chronological (oldest first).
 * Returns every message newest-first, which is what an inverted list expects.
 */
export function flattenPages<T>(pages: { messages: T[] }[]): T[] {
  const out: T[] = [];
  for (const page of pages) {
    for (let i = page.messages.length - 1; i >= 0; i--) {
      const m = page.messages[i];
      if (m !== undefined) out.push(m);
    }
  }
  return out;
}

/** True when the peer's read pointer covers this (server-assigned) message id. */
export function isReadByPeer(messageId: string, peerLastReadId: string): boolean {
  if (!/^\d+$/.test(messageId) || !/^\d+$/.test(peerLastReadId)) return false;
  return BigInt(messageId) <= BigInt(peerLastReadId);
}

/** Id of the most recent message sent by `myId` (list is newest-first). */
export function latestOwnMessageId<T extends WithId & { senderId: string }>(newestFirst: T[], myId: string): string | null {
  return newestFirst.find((m) => m.senderId === myId)?.id ?? null;
}
