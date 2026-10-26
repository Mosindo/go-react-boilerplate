import type { InfiniteData } from "@tanstack/react-query";
import type { Message } from "../api";
import { applyReadReceipt, insertMessage, type MessagePage } from "../lib/chatCache";

const msg = (id: number, senderId: string, readAt: string | null = null): Message => ({
  id,
  matchId: "m1",
  senderId,
  body: `message ${id}`,
  createdAt: "2026-01-01T10:00:00Z",
  readAt
});

const data = (messages: Message[]): InfiniteData<MessagePage, number | undefined> => ({
  pages: [{ messages, hasMore: false }],
  pageParams: [undefined]
});

describe("chat cache updates", () => {
  it("inserts a new message first and ignores duplicates", () => {
    const base = data([msg(1, "a")]);
    const next = insertMessage(base, msg(2, "b"));
    expect(next?.pages[0]?.messages.map((m) => m.id)).toEqual([2, 1]);
    expect(insertMessage(next, msg(2, "b"))).toBe(next);
  });

  it("does nothing when the conversation is not cached", () => {
    expect(insertMessage(undefined, msg(1, "a"))).toBeUndefined();
  });

  it("marks only the other participant's messages as read", () => {
    const base = data([msg(3, "me"), msg(2, "them"), msg(1, "me", "2026-01-01T09:00:00Z")]);
    const next = applyReadReceipt(base, "them", "2026-01-01T11:00:00Z");
    const [m3, m2, m1] = next?.pages[0]?.messages ?? [];
    expect(m3?.readAt).toBe("2026-01-01T11:00:00Z"); // mine, read by them
    expect(m2?.readAt).toBeNull(); // theirs, untouched
    expect(m1?.readAt).toBe("2026-01-01T09:00:00Z"); // already read, preserved
  });
});
