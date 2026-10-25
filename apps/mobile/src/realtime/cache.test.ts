import { conversation, message, matchItem, notification, notificationPages, pages } from "../test-utils";
import {
  appendMessage,
  applyMessageToConversations,
  applyNotification,
  applyReadToConversations,
  clearConversationUnread,
  createLocalId,
  isLocalMessage,
  markAllNotificationsRead,
  markMessagesRead,
  markNotificationRead,
  prependMatch,
  removeConversation,
  removeMessage,
  replaceMessage
} from "./cache";

describe("thread cache", () => {
  it("prepends a new message and ignores duplicates", () => {
    const thread = pages([message({ id: "m1" })]);
    const next = appendMessage(thread, message({ id: "m2" }));
    expect(next?.pages[0]?.items.map((m) => m.id)).toEqual(["m2", "m1"]);
    expect(appendMessage(next, message({ id: "m2" }))).toBe(next);
  });

  it("does nothing when the thread was never loaded", () => {
    expect(appendMessage(undefined, message())).toBeUndefined();
  });

  it("swaps an optimistic message for the server copy", () => {
    const local = message({ id: createLocalId(1), senderId: "me" });
    expect(isLocalMessage(local)).toBe(true);
    const thread = pages([local, message({ id: "m1" })]);
    const next = replaceMessage(thread, local.id, message({ id: "server", senderId: "me" }));
    expect(next?.pages[0]?.items.map((m) => m.id)).toEqual(["server", "m1"]);
  });

  it("drops the optimistic copy when the websocket echo already delivered the real one", () => {
    const local = message({ id: createLocalId(1), senderId: "me" });
    const thread = pages([local, message({ id: "server", senderId: "me" })]);
    const next = replaceMessage(thread, local.id, message({ id: "server", senderId: "me" }));
    expect(next?.pages[0]?.items.map((m) => m.id)).toEqual(["server"]);
    expect(removeMessage(thread, local.id)?.pages[0]?.items).toHaveLength(1);
  });

  it("applies read receipts only to the other side's older unread messages", () => {
    const thread = pages([
      message({ id: "late", senderId: "me", createdAt: "2026-03-10T12:10:00.000Z" }),
      message({ id: "mine", senderId: "me", createdAt: "2026-03-10T11:00:00.000Z" }),
      message({ id: "theirs", senderId: "other", createdAt: "2026-03-10T10:00:00.000Z" })
    ]);
    const next = markMessagesRead(thread, {
      conversationId: "c1",
      readerId: "other",
      readAt: "2026-03-10T12:00:00.000Z"
    });
    const byId = Object.fromEntries((next?.pages[0]?.items ?? []).map((m) => [m.id, m.readAt]));
    expect(byId).toEqual({ late: null, mine: "2026-03-10T12:00:00.000Z", theirs: null });
  });
});

describe("conversation list cache", () => {
  const options = { currentUserId: "me", isActive: false };

  it("updates the preview, moves the conversation first and counts unread", () => {
    const list = pages([conversation({ id: "c2" }), conversation({ id: "c1", unreadCount: 1 })]);
    const incoming = message({ id: "m9", conversationId: "c1", createdAt: "2026-03-11T08:00:00.000Z" });
    const { data, found } = applyMessageToConversations(list, incoming, options);
    expect(found).toBe(true);
    expect(data?.pages[0]?.items.map((c) => c.id)).toEqual(["c1", "c2"]);
    expect(data?.pages[0]?.items[0]).toMatchObject({ unreadCount: 2, lastMessage: { id: "m9" } });
  });

  it("does not count unread for my own messages or for the open thread", () => {
    const list = pages([conversation()]);
    const mine = applyMessageToConversations(list, message({ id: "m2", senderId: "me" }), options);
    expect(mine.data?.pages[0]?.items[0]?.unreadCount).toBe(0);
    const active = applyMessageToConversations(list, message({ id: "m3" }), { ...options, isActive: true });
    expect(active.data?.pages[0]?.items[0]?.unreadCount).toBe(0);
  });

  it("is idempotent for a message it already shows", () => {
    const list = pages([conversation({ lastMessage: message({ id: "m1" }), unreadCount: 1 })]);
    const { data } = applyMessageToConversations(list, message({ id: "m1" }), options);
    expect(data?.pages[0]?.items[0]?.unreadCount).toBe(1);
  });

  it("reports unknown conversations so the caller can refetch", () => {
    expect(
      applyMessageToConversations(pages([conversation()]), message({ conversationId: "zzz" }), options).found
    ).toBe(false);
  });

  it("handles read receipts for either side, removal and unread reset", () => {
    const list = pages([conversation({ unreadCount: 3, lastMessage: message({ senderId: "me" }) })]);
    const byOther = applyReadToConversations(
      list,
      { conversationId: "c1", readerId: "other", readAt: "2026-03-10T13:00:00.000Z" },
      "me"
    );
    expect(byOther?.pages[0]?.items[0]?.lastMessage.readAt).toBe("2026-03-10T13:00:00.000Z");
    const byMe = applyReadToConversations(
      list,
      { conversationId: "c1", readerId: "me", readAt: "2026-03-10T13:00:00.000Z" },
      "me"
    );
    expect(byMe?.pages[0]?.items[0]?.unreadCount).toBe(0);
    expect(clearConversationUnread(list, "c1")?.pages[0]?.items[0]?.unreadCount).toBe(0);
    expect(removeConversation(list, "c1")?.pages[0]?.items).toEqual([]);
  });
});

describe("matches and notifications", () => {
  it("prepends a match once", () => {
    const list = pages([matchItem({ matchId: "a" })]);
    const next = prependMatch(list, matchItem({ matchId: "b" }));
    expect(next?.pages[0]?.items.map((m) => m.matchId)).toEqual(["b", "a"]);
    expect(prependMatch(next, matchItem({ matchId: "b" }))).toBe(next);
  });

  it("adds notifications and bumps the unread count", () => {
    const data = notificationPages([notification({ id: "old", type: "match", conversationId: "c9" })], 1);
    const next = applyNotification(data, notification({ id: "new", type: "match", conversationId: "c2" }));
    expect(next?.pages[0]?.items.map((n) => n.id)).toEqual(["new", "old"]);
    expect(next?.pages[0]?.unreadCount).toBe(2);
  });

  it("coalesces unread message notifications of the same conversation", () => {
    const data = notificationPages([notification({ id: "old" })], 1);
    const next = applyNotification(data, notification({ id: "newer" }));
    expect(next?.pages[0]?.items.map((n) => n.id)).toEqual(["newer"]);
    expect(next?.pages[0]?.unreadCount).toBe(1);
  });

  it("marks one or all notifications read", () => {
    const data = notificationPages([notification({ id: "a" }), notification({ id: "b", conversationId: "c2" })], 2);
    const one = markNotificationRead(data, "a", "2026-03-10T13:00:00.000Z");
    expect(one?.pages[0]?.unreadCount).toBe(1);
    expect(markNotificationRead(one, "a", "2026-03-10T14:00:00.000Z")).toBe(one);
    const all = markAllNotificationsRead(data, "2026-03-10T13:00:00.000Z");
    expect(all?.pages[0]?.unreadCount).toBe(0);
    expect(all?.pages[0]?.items.every((n) => n.readAt !== null)).toBe(true);
  });
});
