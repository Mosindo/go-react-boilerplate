import { describe, expect, it } from "vitest";
import type { Message } from "../api/types";
import { insertMessage, markMessagesRead, type MessagePages } from "./cache";

const msg = (id: string, senderId: string, readAt?: string): Message => ({
  id,
  conversationId: "c1",
  senderId,
  content: id,
  createdAt: "2026-01-01T10:00:00Z",
  readAt,
});

const pages = (...items: Message[][]): MessagePages => ({
  pages: items,
  pageParams: items.map(() => undefined),
});

describe("insertMessage", () => {
  it("prepends a new message to the newest page", () => {
    const next = insertMessage(pages([msg("a", "u1")], [msg("old", "u1")]), msg("b", "u2"));
    expect(next?.pages[0]?.map((m) => m.id)).toEqual(["b", "a"]);
    expect(next?.pages[1]?.map((m) => m.id)).toEqual(["old"]);
  });

  it("ignores duplicates (HTTP response + websocket event)", () => {
    const data = pages([msg("a", "u1")]);
    expect(insertMessage(data, msg("a", "u1"))).toBe(data);
  });

  it("does nothing when the conversation is not cached", () => {
    expect(insertMessage(undefined, msg("a", "u1"))).toBeUndefined();
  });
});

describe("markMessagesRead", () => {
  it("flags only the messages the reader received", () => {
    const next = markMessagesRead(
      pages([msg("mine", "reader"), msg("theirs", "other")]),
      "reader",
      "T",
    );
    const [mine, theirs] = next?.pages[0] ?? [];
    expect(mine?.readAt).toBeUndefined();
    expect(theirs?.readAt).toBe("T");
  });
});
