import type { AppNotification, ChatMessage, MatchSummary, Message } from "../../api/types";
import {
  addMatch,
  addMessage,
  addNotification,
  applyMessageToMatches,
  clearUnread,
  markAllNotificationsReadInCache,
  markNotificationReadInCache,
  markOutgoingRead,
  removeMatch,
  replaceMessage,
  setMessageStatus,
  totalUnread,
  type MatchPages,
  type MessagePages,
  type NotificationPages
} from "../realtimeCache";

function match(id: string, extra: Partial<MatchSummary> = {}): MatchSummary {
  return {
    matchId: `m-${id}`,
    conversationId: `c-${id}`,
    createdAt: "2026-01-01T00:00:00Z",
    user: { userId: `u-${id}`, firstName: id, age: 30, photo: null },
    lastMessage: null,
    unreadCount: 0,
    ...extra
  };
}

function matchPages(...pages: MatchSummary[][]): MatchPages {
  return {
    pages: pages.map((items) => ({ items, nextCursor: null })),
    pageParams: pages.map(() => null)
  };
}

function message(id: string, extra: Partial<ChatMessage> = {}): ChatMessage {
  return {
    id,
    conversationId: "c-a",
    senderId: "them",
    body: id,
    createdAt: "2026-01-01T10:00:00Z",
    readAt: null,
    ...extra
  };
}

function messagePages(...items: ChatMessage[]): MessagePages {
  return { pages: [{ items, nextCursor: null }], pageParams: [null] };
}

describe("matches cache", () => {
  it("moves the conversation to the top with a preview and counts incoming unread", () => {
    const data = matchPages([match("a"), match("b")]);
    const incoming: Message = message("x", { conversationId: "c-b", body: "hey" });
    const next = applyMessageToMatches(data, incoming, "me", null);
    expect(next?.pages[0].items.map((item) => item.matchId)).toEqual(["m-b", "m-a"]);
    expect(next?.pages[0].items[0].lastMessage?.body).toBe("hey");
    expect(next?.pages[0].items[0].unreadCount).toBe(1);
  });

  it("does not count own messages or messages of the open conversation as unread", () => {
    const data = matchPages([match("a")]);
    const own = applyMessageToMatches(data, message("x", { senderId: "me" }), "me", null);
    expect(own?.pages[0].items[0].unreadCount).toBe(0);
    const open = applyMessageToMatches(data, message("y"), "me", "c-a");
    expect(open?.pages[0].items[0].unreadCount).toBe(0);
  });

  it("leaves the cache untouched for unknown conversations or a missing cache", () => {
    const data = matchPages([match("a")]);
    expect(applyMessageToMatches(data, message("x", { conversationId: "c-zzz" }), "me", null)).toBe(data);
    expect(applyMessageToMatches(undefined, message("x"), "me", null)).toBeUndefined();
  });

  it("clears unread and totals across pages", () => {
    const data = matchPages([match("a", { unreadCount: 2 })], [match("b", { unreadCount: 3 })]);
    expect(totalUnread(data)).toBe(5);
    expect(totalUnread(clearUnread(data, "c-a"))).toBe(3);
    expect(totalUnread(undefined)).toBe(0);
  });

  it("adds a new match once and removes a match", () => {
    const data = matchPages([match("a")]);
    const added = addMatch(data, match("b"));
    expect(added?.pages[0].items.map((item) => item.matchId)).toEqual(["m-b", "m-a"]);
    expect(addMatch(added, match("b"))).toBe(added);
    expect(removeMatch(added, "m-a")?.pages[0].items.map((item) => item.matchId)).toEqual(["m-b"]);
  });
});

describe("messages cache", () => {
  it("adds a message on top and deduplicates by id", () => {
    const data = messagePages(message("old"));
    const added = addMessage(data, message("new"));
    expect(added?.pages[0].items.map((item) => item.id)).toEqual(["new", "old"]);
    const again = addMessage(added, message("new", { readAt: "2026-01-01T11:00:00Z" }));
    expect(again?.pages[0].items).toHaveLength(2);
    expect(again?.pages[0].items[0].readAt).not.toBeNull();
  });

  it("replaces an optimistic message with the saved one", () => {
    const data = messagePages(message("local-1", { status: "sending", senderId: "me" }));
    const saved = message("real-1", { senderId: "me" });
    const next = replaceMessage(data, "local-1", saved);
    expect(next?.pages[0].items).toEqual([saved]);
  });

  it("drops the optimistic copy when realtime already delivered the saved message", () => {
    const saved = message("real-1", { senderId: "me" });
    const data = messagePages(saved, message("local-1", { status: "sending", senderId: "me" }));
    expect(replaceMessage(data, "local-1", saved)?.pages[0].items).toEqual([saved]);
  });

  it("marks a message as failed", () => {
    const data = messagePages(message("local-1", { status: "sending" }));
    expect(setMessageStatus(data, "local-1", "failed")?.pages[0].items[0].status).toBe("failed");
  });

  it("marks only my confirmed outgoing messages as read", () => {
    const data = messagePages(
      message("mine", { senderId: "me" }),
      message("pending", { senderId: "me", status: "sending" }),
      message("theirs")
    );
    const next = markOutgoingRead(data, "me", "2026-01-02T00:00:00Z");
    const byId = Object.fromEntries((next?.pages[0].items ?? []).map((item) => [item.id, item.readAt]));
    expect(byId).toEqual({ mine: "2026-01-02T00:00:00Z", pending: null, theirs: null });
  });
});

describe("notifications cache", () => {
  function notification(id: string, readAt: string | null = null): AppNotification {
    return { id, type: "match", title: "t", body: "b", data: {}, readAt, createdAt: "2026-01-01T00:00:00Z" };
  }
  function pages(unreadCount: number, ...items: AppNotification[]): NotificationPages {
    return { pages: [{ items, nextCursor: null, unreadCount }], pageParams: [null] };
  }

  it("prepends new notifications, bumps the unread count and ignores duplicates", () => {
    const data = pages(1, notification("a"));
    const next = addNotification(data, notification("b"));
    expect(next?.pages[0].items.map((item) => item.id)).toEqual(["b", "a"]);
    expect(next?.pages[0].unreadCount).toBe(2);
    expect(addNotification(next, notification("b"))).toBe(next);
  });

  it("marks one notification read and decrements the badge once", () => {
    const data = pages(2, notification("a"), notification("b"));
    const next = markNotificationReadInCache(data, "a", "2026-01-02T00:00:00Z");
    expect(next?.pages[0].unreadCount).toBe(1);
    expect(next?.pages[0].items[0].readAt).not.toBeNull();
    expect(markNotificationReadInCache(next, "a", "2026-01-03T00:00:00Z")).toBe(next);
  });

  it("marks everything read", () => {
    const data = pages(2, notification("a"), notification("b"));
    const next = markAllNotificationsReadInCache(data, "2026-01-02T00:00:00Z");
    expect(next?.pages[0].unreadCount).toBe(0);
    expect(next?.pages[0].items.every((item) => item.readAt !== null)).toBe(true);
  });
});
