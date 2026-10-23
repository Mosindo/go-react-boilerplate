import type { Message } from "../api/types";

export type PendingMessage = {
  localId: string;
  content: string;
  createdAt: string;
  status: "sending" | "failed";
};

export type ChatEntry =
  { kind: "message"; message: Message } | { kind: "pending"; pending: PendingMessage };

let counter = 0;
export function newLocalId(): string {
  counter += 1;
  return `local-${Date.now()}-${counter}`;
}

/** Newest-first list for an inverted FlatList: pending messages sit on top of the server history. */
export function buildChatEntries(messages: Message[], pending: PendingMessage[]): ChatEntry[] {
  const newestFirst = [...pending].reverse();
  return [
    ...newestFirst.map((item) => ({ kind: "pending", pending: item }) as const),
    ...messages.map((message) => ({ kind: "message", message }) as const)
  ];
}

/** Id of my most recent message (the one that carries the sent/read receipt). */
export function lastOwnMessageId(messages: Message[], myUserId: string | null): string | null {
  return messages.find((message) => message.senderId === myUserId)?.id ?? null;
}
