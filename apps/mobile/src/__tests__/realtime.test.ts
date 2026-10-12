import type { InfiniteData } from "@tanstack/react-query";
import type { ChatMessage, MessagePage } from "../api/types";
import { appendMessage } from "../hooks/useRealtime";

const msg = (id: string): ChatMessage => ({ id, matchId: "m1", senderId: "u1", body: id, createdAt: "2026-01-01T00:00:00Z" });
const cache = (items: ChatMessage[]): InfiniteData<MessagePage, string | undefined> => ({
  pages: [{ items, nextBefore: "", otherLastReadAt: null }],
  pageParams: [undefined]
});

describe("appendMessage", () => {
  it("puts the newest message first", () => {
    const next = appendMessage(cache([msg("a")]), msg("b"));
    expect(next?.pages[0].items.map((m) => m.id)).toEqual(["b", "a"]);
  });

  it("ignores duplicates delivered by both the HTTP response and the socket", () => {
    const base = cache([msg("a")]);
    expect(appendMessage(base, msg("a"))).toBe(base);
  });

  it("does nothing when the conversation is not loaded", () => {
    expect(appendMessage(undefined, msg("a"))).toBeUndefined();
  });
});
