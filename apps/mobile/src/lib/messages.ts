import type { Message } from "../api/types";

/** Inserts a message in a newest-first list unless it is already there (REST reply and WebSocket event can both deliver it). */
export function mergeMessage(newestFirst: Message[], incoming: Message): Message[] {
  if (newestFirst.some((m) => m.id === incoming.id)) return newestFirst;
  return [incoming, ...newestFirst].sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : a.createdAt > b.createdAt ? -1 : 0
  );
}

/** True when the message was sent by `me` and the peer has read it. */
export function isRead(message: Message, meId: string, peerLastReadAt: string): boolean {
  return (
    message.senderId === meId && new Date(peerLastReadAt).getTime() >= new Date(message.createdAt).getTime()
  );
}
