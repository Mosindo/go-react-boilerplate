import type { Message } from "../api/types";
import { appendMessage, flattenMessages, markReadBy, type MessagesCache } from "./messageCache";

const msg = (id: number, senderId: string, readAt: string | null = null): Message => ({
  id,
  conversationId: "c1",
  senderId,
  body: `m${id}`,
  createdAt: "2026-01-01T10:00:00Z",
  readAt
});

const cache = (): MessagesCache => ({
  pageParams: [undefined, 3],
  pages: [
    { messages: [msg(4, "a"), msg(5, "b")], nextBefore: 4 },
    { messages: [msg(1, "a"), msg(2, "b"), msg(3, "a")], nextBefore: null }
  ]
});

describe("message cache", () => {
  it("appends new messages to the newest page once", () => {
    const next = appendMessage(cache(), msg(6, "a"));
    expect(next?.pages[0]?.messages.map((m) => m.id)).toEqual([4, 5, 6]);
    expect(appendMessage(next, msg(6, "a"))).toBe(next);
    expect(appendMessage(undefined, msg(7, "a"))).toBeUndefined();
  });

  it("flattens pages oldest first", () => {
    expect(flattenMessages(cache()).map((m) => m.id)).toEqual([1, 2, 3, 4, 5]);
  });

  it("marks the other side's messages as read", () => {
    const next = markReadBy(cache(), "b", "2026-01-01T11:00:00Z");
    const flat = flattenMessages(next);
    expect(flat.filter((m) => m.senderId === "a").every((m) => m.readAt !== null)).toBe(true);
    expect(flat.filter((m) => m.senderId === "b").every((m) => m.readAt === null)).toBe(true);
  });
});
