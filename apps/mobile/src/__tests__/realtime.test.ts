import { QueryClient, type InfiniteData } from "@tanstack/react-query";

import type { Message, MessagePage } from "../api/types";
import { applyEvent, keys, markMessagesRead, mergeMessage } from "../realtime/RealtimeProvider";

const msg = (id: number, senderId: string, readAt?: string): Message => ({
  id,
  matchId: "m1",
  senderId,
  body: `msg ${id}`,
  createdAt: "2026-06-15T10:00:00Z",
  readAt,
});

const cache = (...messages: Message[]): InfiniteData<MessagePage> => ({
  pages: [{ messages, hasMore: false }],
  pageParams: [undefined],
});

describe("mergeMessage", () => {
  it("appends new messages to the newest page", () => {
    const next = mergeMessage(cache(msg(1, "a")), msg(2, "b"));
    expect(next?.pages[0]?.messages.map((m) => m.id)).toEqual([1, 2]);
  });

  it("ignores duplicates delivered by both REST and WebSocket", () => {
    const data = cache(msg(1, "a"));
    expect(mergeMessage(data, msg(1, "a"))).toBe(data);
  });

  it("does nothing when the conversation is not cached", () => {
    expect(mergeMessage(undefined, msg(1, "a"))).toBeUndefined();
  });
});

describe("read receipts", () => {
  it("marks only my own messages as read when the other person reads", () => {
    const next = markMessagesRead(cache(msg(1, "me"), msg(2, "them")), true, "me");
    const [mine, theirs] = next?.pages[0]?.messages ?? [];
    expect(mine?.readAt).toBeDefined();
    expect(theirs?.readAt).toBeUndefined();
  });
});

describe("applyEvent", () => {
  it("updates the conversation cache and refreshes counters on message", () => {
    const qc = new QueryClient();
    qc.setQueryData(keys.messages("m1"), cache(msg(1, "them")));
    qc.setQueryData(keys.summary, { unreadMessages: 0, unreadNotifications: 0 });
    applyEvent(qc, { type: "message", payload: msg(2, "them") }, "me");
    const data = qc.getQueryData<InfiniteData<MessagePage>>(keys.messages("m1"));
    expect(data?.pages[0]?.messages).toHaveLength(2);
    expect(qc.getQueryState(keys.summary)?.isInvalidated).toBe(true);
  });

  it("drops the cached conversation when unmatched", () => {
    const qc = new QueryClient();
    qc.setQueryData(keys.messages("m1"), cache(msg(1, "them")));
    applyEvent(qc, { type: "unmatched", payload: { matchId: "m1" } }, "me");
    expect(qc.getQueryData(keys.messages("m1"))).toBeUndefined();
  });
});
