import { describe, expect, it } from "vitest";
import type { AppNotification, ConversationSummary, Message } from "../../api/models";
import {
  applyMessageToConversations,
  findConversation,
  markAllNotificationsReadInCache,
  markConversationLastMessageRead,
  markNotificationReadInCache,
  prependNotification,
  putConversationFirst,
  removeConversation,
  selectActiveConversations,
  selectNewMatches,
  setConversationUnread,
  totalUnreadMessages,
  unreadNotificationCount,
  flattenConversations,
  type ConversationsPage,
  type NotificationsPage
} from "./cache";

const me = "me";

function message(id: string, conversationId: string, senderId: string, at: string): Message {
  return { id, conversationId, senderId, body: id, createdAt: at, readAt: null };
}

function conv(id: string, updatedAt: string, unread = 0, last: Message | null = null): ConversationSummary {
  return {
    id,
    matchId: `match-${id}`,
    matchedAt: updatedAt,
    user: { userId: `u-${id}`, firstName: id, age: 30, photo: null },
    lastMessage: last,
    unreadCount: unread,
    updatedAt
  };
}

function convData(...pages: ConversationSummary[][]): { pages: ConversationsPage[] } {
  return { pages: pages.map((conversations) => ({ conversations })) };
}

describe("conversation reducers", () => {
  it("moves a conversation to the top on an incoming message and bumps unread", () => {
    const data = convData([conv("a", "2026-01-01T00:00:00Z"), conv("b", "2026-01-02T00:00:00Z")], [
      conv("c", "2025-12-01T00:00:00Z")
    ]);
    const m = message("m1", "c", "them", "2026-01-03T00:00:00Z");
    const out = applyMessageToConversations(data, m, { myUserId: me, isOpen: false });
    expect(out.found).toBe(true);
    expect(flattenConversations(out.data.pages).map((c) => c.id)).toEqual(["c", "a", "b"]);
    expect(findConversation(out.data.pages, "c")?.unreadCount).toBe(1);
    expect(findConversation(out.data.pages, "c")?.lastMessage?.id).toBe("m1");
  });

  it("does not bump unread when the thread is open, or for my own message, or a duplicate", () => {
    const m = message("m1", "a", "them", "2026-01-03T00:00:00Z");
    const data = convData([conv("a", "2026-01-01T00:00:00Z")]);
    const open = applyMessageToConversations(data, m, { myUserId: me, isOpen: true });
    expect(findConversation(open.data.pages, "a")?.unreadCount).toBe(0);
    const mine = applyMessageToConversations(data, { ...m, senderId: me }, { myUserId: me, isOpen: false });
    expect(findConversation(mine.data.pages, "a")?.unreadCount).toBe(0);
    const first = applyMessageToConversations(data, m, { myUserId: me, isOpen: false });
    const dup = applyMessageToConversations(first.data, m, { myUserId: me, isOpen: false });
    expect(findConversation(dup.data.pages, "a")?.unreadCount).toBe(1);
  });

  it("reports unknown conversations", () => {
    const out = applyMessageToConversations(convData([]), message("m", "zzz", "them", "2026-01-01T00:00:00Z"), {
      myUserId: me,
      isOpen: false
    });
    expect(out.found).toBe(false);
  });

  it("prepends, replaces and removes", () => {
    let data = convData([conv("a", "2026-01-01T00:00:00Z")]);
    data = putConversationFirst(data, conv("b", "2026-01-02T00:00:00Z"));
    data = putConversationFirst(data, conv("a", "2026-01-03T00:00:00Z"));
    expect(flattenConversations(data.pages).map((c) => c.id)).toEqual(["a", "b"]);
    data = removeConversation(data, "a");
    expect(flattenConversations(data.pages).map((c) => c.id)).toEqual(["b"]);
    expect(removeConversation(data, "nope")).toBe(data);
  });

  it("sets unread and sums it", () => {
    let data = convData([conv("a", "2026-01-01T00:00:00Z", 2), conv("b", "2026-01-01T00:00:00Z", 3)]);
    expect(totalUnreadMessages(data.pages)).toBe(5);
    data = setConversationUnread(data, "a", 0);
    expect(totalUnreadMessages(data.pages)).toBe(3);
    expect(totalUnreadMessages(undefined)).toBe(0);
  });

  it("flags my last message as read", () => {
    const last = message("m1", "a", me, "2026-01-01T00:00:00Z");
    const data = convData([conv("a", "2026-01-01T00:00:00Z", 0, last)]);
    const out = markConversationLastMessageRead(data, "a", me, "2026-01-02T00:00:00Z");
    expect(findConversation(out.pages, "a")?.lastMessage?.readAt).toBe("2026-01-02T00:00:00Z");
  });

  it("splits new matches from active conversations", () => {
    const list = [conv("a", "2026-01-01T00:00:00Z"), conv("b", "2026-01-01T00:00:00Z", 0, message("m", "b", me, "x"))];
    expect(selectNewMatches(list).map((c) => c.id)).toEqual(["a"]);
    expect(selectActiveConversations(list).map((c) => c.id)).toEqual(["b"]);
  });
});

function notif(id: string, isRead = false): AppNotification {
  return { id, type: "message", title: "t", body: "b", data: {}, isRead, createdAt: "2026-01-01T00:00:00Z" };
}

describe("notification reducers", () => {
  const base: { pages: NotificationsPage[] } = {
    pages: [{ notifications: [notif("n1"), notif("n2", true)], unreadCount: 1 }]
  };

  it("prepends once and bumps unread", () => {
    const out = prependNotification(base, notif("n3"));
    expect(out.pages[0].notifications.map((n) => n.id)).toEqual(["n3", "n1", "n2"]);
    expect(unreadNotificationCount(out.pages)).toBe(2);
    expect(prependNotification(out, notif("n3"))).toBe(out);
  });

  it("marks one read", () => {
    const out = markNotificationReadInCache(base, "n1");
    expect(unreadNotificationCount(out.pages)).toBe(0);
    expect(markNotificationReadInCache(out, "n1")).toBe(out);
    expect(markNotificationReadInCache(base, "n2")).toBe(base);
  });

  it("marks all read", () => {
    const out = markAllNotificationsReadInCache(base);
    expect(unreadNotificationCount(out.pages)).toBe(0);
    expect(out.pages[0].notifications.every((n) => n.isRead)).toBe(true);
  });
});
