import { describe, expect, it } from "vitest";
import type { Message } from "../api/types";
import { flattenMessages, insertMessage, type MessagePages } from "./messages";

const msg = (id: number): Message => ({
  id,
  conversationId: "c",
  senderId: "u",
  body: `m${id}`,
  createdAt: "2026-01-01T00:00:00Z",
  read: false
});

const pages = (): MessagePages => ({
  pages: [
    { messages: [msg(3), msg(2)], hasMore: true },
    { messages: [msg(1)], hasMore: false }
  ],
  pageParams: [undefined, 2]
});

describe("insertMessage", () => {
  it("prepends to the newest page", () => {
    const out = insertMessage(pages(), msg(4));
    expect(flattenMessages(out).map((m) => m.id)).toEqual([4, 3, 2, 1]);
  });
  it("ignores duplicates", () => {
    const data = pages();
    expect(insertMessage(data, msg(2))).toBe(data);
  });
  it("leaves an unloaded cache untouched", () => {
    expect(insertMessage(undefined, msg(1))).toBeUndefined();
  });
});
