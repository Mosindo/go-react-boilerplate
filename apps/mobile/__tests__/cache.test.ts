import type { Conversation, Message, AppNotification } from "../src/api/types";
import {
  applyIncomingMessage,
  applyReadReceipt,
  clearUnread,
  markNotificationsRead,
  parseRealtimeEvent,
  prependConversation,
  prependNotification,
  removeConversation,
  totalUnread,
  upsertMessage,
  type ConversationsCache,
  type MessagesCache,
  type NotificationsCache
} from "../src/realtime/cache";

const msg = (id: string, senderId: string, extra: Partial<Message> = {}): Message => ({
  id,
  conversationId: "c1",
  senderId,
  body: id,
  createdAt: "2026-01-01T00:00:00Z",
  readAt: null,
  ...extra
});
const conv = (id: string, unread = 0): Conversation => ({
  id,
  matchedAt: "2026-01-01T00:00:00Z",
  user: { userId: `u-${id}`, firstName: id, age: 30, photoUrl: null },
  lastMessage: null,
  unreadCount: unread
});
const convCache = (...c: Conversation[]): ConversationsCache => ({
  pages: [{ conversations: c, nextCursor: null }],
  pageParams: [null]
});
const msgCache = (...m: Message[]): MessagesCache => ({ pages: [{ messages: m, nextCursor: null }], pageParams: [null] });

describe("messages cache", () => {
  it("prepends new messages and dedupes by id", () => {
    const cache = msgCache(msg("m1", "a"));
    const added = upsertMessage(cache, msg("m2", "b"));
    expect(added?.pages[0]?.messages.map((m) => m.id)).toEqual(["m2", "m1"]);
    const again = upsertMessage(added, msg("m2", "b", { readAt: "x" }));
    expect(again?.pages[0]?.messages).toHaveLength(2);
    expect(again?.pages[0]?.messages[0]?.readAt).toBe("x");
  });
  it("ignores an uncached conversation", () => {
    expect(upsertMessage(undefined, msg("m1", "a"))).toBeUndefined();
  });
  it("marks the other side's messages read on a receipt", () => {
    const cache = msgCache(msg("m2", "me"), msg("m1", "other"));
    const out = applyReadReceipt(cache, "other", "T");
    expect(out?.pages[0]?.messages.find((m) => m.id === "m2")?.readAt).toBe("T");
    expect(out?.pages[0]?.messages.find((m) => m.id === "m1")?.readAt).toBeNull();
  });
});

describe("conversations cache", () => {
  it("moves the conversation to the top and counts unread for incoming messages", () => {
    const cache = convCache(conv("c0"), conv("c1"));
    const res = applyIncomingMessage(cache, msg("m1", "other"), "me", null);
    expect(res.found).toBe(true);
    expect(res.cache?.pages[0]?.conversations.map((c) => c.id)).toEqual(["c1", "c0"]);
    expect(res.cache?.pages[0]?.conversations[0]?.unreadCount).toBe(1);
  });
  it("does not count unread for own or active-conversation messages", () => {
    const cache = convCache(conv("c1"));
    expect(applyIncomingMessage(cache, msg("m1", "me"), "me", null).cache?.pages[0]?.conversations[0]?.unreadCount).toBe(0);
    expect(applyIncomingMessage(cache, msg("m1", "other"), "me", "c1").cache?.pages[0]?.conversations[0]?.unreadCount).toBe(0);
  });
  it("reports unknown conversations so the caller can refetch", () => {
    expect(applyIncomingMessage(convCache(conv("c9")), msg("m1", "other"), "me", null).found).toBe(false);
  });
  it("prepends matches without duplicating, clears unread, removes", () => {
    const cache = convCache(conv("c1", 3));
    expect(prependConversation(cache, conv("c1"))).toBe(cache);
    expect(prependConversation(cache, conv("c2"))?.pages[0]?.conversations[0]?.id).toBe("c2");
    expect(totalUnread(cache)).toBe(3);
    expect(totalUnread(clearUnread(cache, "c1"))).toBe(0);
    expect(removeConversation(cache, "c1")?.pages[0]?.conversations).toHaveLength(0);
  });
});

describe("notifications cache", () => {
  const n = (id: string, readAt: string | null = null): AppNotification => ({
    id,
    type: "match",
    conversationId: "c1",
    actor: { userId: "u", firstName: "A" },
    readAt,
    createdAt: "2026-01-01T00:00:00Z"
  });
  const cache: NotificationsCache = { pages: [{ notifications: [n("n1")], nextCursor: null, unreadCount: 1 }], pageParams: [null] };

  it("prepends and counts unread", () => {
    const out = prependNotification(cache, n("n2"));
    expect(out?.pages[0]?.unreadCount).toBe(2);
    expect(prependNotification(out, n("n2"))).toBe(out);
  });
  it("marks one or all read", () => {
    expect(markNotificationsRead(cache, "n1")?.pages[0]?.unreadCount).toBe(0);
    expect(markNotificationsRead(cache, "all")?.pages[0]?.notifications[0]?.readAt).not.toBeNull();
  });
});

describe("parseRealtimeEvent", () => {
  it("accepts known events and rejects junk", () => {
    expect(parseRealtimeEvent(JSON.stringify({ type: "message", data: msg("m1", "a") }))?.type).toBe("message");
    expect(parseRealtimeEvent(JSON.stringify({ type: "weird", data: {} }))).toBeNull();
    expect(parseRealtimeEvent("{not json")).toBeNull();
  });
});
