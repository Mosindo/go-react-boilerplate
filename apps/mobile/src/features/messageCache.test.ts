import { insertMessage } from "./messageCache";
import type { Message } from "../api/types";

const msg = (id: string): Message => ({
  id,
  conversationId: "c",
  senderId: "u",
  body: id,
  createdAt: "2026-01-01T00:00:00Z",
  readAt: null
});

describe("insertMessage", () => {
  const base = { pages: [[msg("2"), msg("1")]], pageParams: [undefined] };

  it("puts new messages first", () => {
    const next = insertMessage(base, msg("3"));
    expect(next?.pages[0].map((m) => m.id)).toEqual(["3", "2", "1"]);
  });
  it("ignores duplicates (own message echoed by the socket)", () => {
    expect(insertMessage(base, msg("2"))).toBe(base);
  });
  it("leaves an unloaded conversation untouched", () => {
    expect(insertMessage(undefined, msg("1"))).toBeUndefined();
  });
});
