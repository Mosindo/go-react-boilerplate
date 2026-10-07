import {
  applyMatchToMatches,
  applyMessageToConversations,
  applyMessageToPages,
  applyNotification,
  applyReadToConversations,
  buildSocketUrl,
  clearConversationUnread,
  markAllReadInPayload,
  markNotificationReadInPayload,
  newMatchesWithoutMessages,
  parseRealtimeEvent,
  reconnectDelay,
  removeMatchesForUser,
  totalUnread
} from "./events";
import type { AppNotification, Conversation, Match, Message } from "../features/types";

const user = { userId: "u2", firstName: "Sam", age: 30, photo: null };

function message(id: string, senderId: string, createdAt: string, conversationId = "c1"): Message {
  return { id, conversationId, senderId, content: id, createdAt };
}

function conversation(
  id: string,
  updatedAt: string,
  unreadCount = 0,
  lastMessage: Message | null = null
): Conversation {
  return { id, matchId: `m-${id}`, user, lastMessage, unreadCount, peerLastReadAt: null, updatedAt };
}

function notification(id: string, isRead = false): AppNotification {
  return { id, type: "message", title: "t", body: "b", data: {}, isRead, createdAt: "2026-10-07T10:00:00Z" };
}

describe("parseRealtimeEvent", () => {
  it("parses well-formed events", () => {
    const frame = JSON.stringify({ type: "message", data: message("a", "u2", "2026-10-07T10:00:00Z") });
    expect(parseRealtimeEvent(frame)?.type).toBe("message");
    const read = JSON.stringify({
      type: "read",
      data: { conversationId: "c1", readerId: "u2", readAt: "2026-10-07T10:00:00Z" }
    });
    expect(parseRealtimeEvent(read)?.type).toBe("read");
  });
  it("rejects pong, junk, and malformed payloads", () => {
    expect(parseRealtimeEvent('{"type":"pong"}')).toBeNull();
    expect(parseRealtimeEvent("not json")).toBeNull();
    expect(parseRealtimeEvent('{"type":"message","data":{"id":1}}')).toBeNull();
    expect(parseRealtimeEvent("[]")).toBeNull();
  });
  it("normalises missing notification data", () => {
    const frame = JSON.stringify({
      type: "notification",
      data: { id: "n", type: "match", title: "", body: "", isRead: false, createdAt: "x" }
    });
    const event = parseRealtimeEvent(frame);
    expect(event?.type === "notification" && event.data.data).toEqual({});
  });
});

describe("socket url and backoff", () => {
  it("maps http to ws and https to wss", () => {
    expect(buildSocketUrl("http://10.0.0.2:18080/", "t k")).toBe("ws://10.0.0.2:18080/ws?ticket=t%20k");
    expect(buildSocketUrl("https://api.example.com", "abc")).toBe("wss://api.example.com/ws?ticket=abc");
  });
  it("grows exponentially and caps", () => {
    expect(reconnectDelay(0)).toBe(1000);
    expect(reconnectDelay(1)).toBe(2000);
    expect(reconnectDelay(3)).toBe(8000);
    expect(reconnectDelay(20)).toBe(30000);
    expect(reconnectDelay(0, 0.5)).toBe(1125);
  });
});

describe("conversation reducers", () => {
  const list = [
    conversation("c1", "2026-10-07T09:00:00Z"),
    conversation("c2", "2026-10-07T08:00:00Z", 1, message("old", "u3", "2026-10-07T08:00:00Z", "c2"))
  ];
  it("moves the conversation to the top and counts unread from the peer", () => {
    const { conversations, known } = applyMessageToConversations(
      list,
      message("n", "u2", "2026-10-07T10:00:00Z", "c2"),
      "me",
      null
    );
    expect(known).toBe(true);
    expect(conversations.map((c) => c.id)).toEqual(["c2", "c1"]);
    expect(conversations[0].unreadCount).toBe(2);
    expect(conversations[0].lastMessage?.id).toBe("n");
  });
  it("does not count my own or active-chat messages", () => {
    const own = applyMessageToConversations(
      list,
      message("n", "me", "2026-10-07T10:00:00Z", "c2"),
      "me",
      null
    );
    expect(own.conversations[0].unreadCount).toBe(1);
    const active = applyMessageToConversations(
      list,
      message("n", "u2", "2026-10-07T10:00:00Z", "c2"),
      "me",
      "c2"
    );
    expect(active.conversations[0].unreadCount).toBe(1);
  });
  it("is idempotent for the same message and flags unknown conversations", () => {
    const first = applyMessageToConversations(
      list,
      message("n", "u2", "2026-10-07T10:00:00Z", "c2"),
      "me",
      null
    ).conversations;
    const second = applyMessageToConversations(
      first,
      message("n", "u2", "2026-10-07T10:00:00Z", "c2"),
      "me",
      null
    );
    expect(second.conversations).toBe(first);
    expect(
      applyMessageToConversations(list, message("n", "u2", "2026-10-07T10:00:00Z", "zzz"), "me", null).known
    ).toBe(false);
  });
  it("applies peer read receipts and own reads", () => {
    const peer = applyReadToConversations(
      list,
      { conversationId: "c1", readerId: "u2", readAt: "2026-10-07T09:30:00Z" },
      "me"
    );
    expect(peer[0].peerLastReadAt).toBe("2026-10-07T09:30:00Z");
    expect(
      applyReadToConversations(
        peer,
        { conversationId: "c1", readerId: "u2", readAt: "2026-10-07T09:00:00Z" },
        "me"
      )
    ).toBe(peer);
    const own = applyReadToConversations(
      list,
      { conversationId: "c2", readerId: "me", readAt: "2026-10-07T09:30:00Z" },
      "me"
    );
    expect(own[1].unreadCount).toBe(0);
    expect(clearConversationUnread(list, "c2")[1].unreadCount).toBe(0);
  });
  it("sums unread", () => {
    expect(totalUnread(list)).toBe(1);
    expect(totalUnread(undefined)).toBe(0);
  });
});

describe("messages page reducer", () => {
  const page = { items: [message("b", "u2", "2026-10-07T10:00:00Z")], nextCursor: null };
  const data = { pages: [page], pageParams: [undefined] };
  it("prepends to the first page once", () => {
    const next = applyMessageToPages(data, message("c", "u2", "2026-10-07T11:00:00Z"));
    expect(next.pages[0].items.map((m) => m.id)).toEqual(["c", "b"]);
    expect(applyMessageToPages(next, message("c", "u2", "2026-10-07T11:00:00Z"))).toBe(next);
  });
  it("ignores empty caches", () => {
    const empty = { pages: [], pageParams: [] };
    expect(applyMessageToPages(empty, message("c", "u2", "2026-10-07T11:00:00Z"))).toBe(empty);
  });
});

describe("matches and notifications reducers", () => {
  const m1: Match = { id: "m1", user, conversationId: "c1", createdAt: "2026-10-07T09:00:00Z" };
  it("adds a match once and finds new matches without messages", () => {
    const list = applyMatchToMatches([], m1);
    expect(applyMatchToMatches(list, m1)).toBe(list);
    const convs = [conversation("c1", "2026-10-07T09:00:00Z")];
    expect(newMatchesWithoutMessages(list, convs)).toHaveLength(1);
    const withMsg = [
      conversation("c1", "2026-10-07T09:00:00Z", 0, message("x", "u2", "2026-10-07T09:10:00Z"))
    ];
    expect(newMatchesWithoutMessages(list, withMsg)).toHaveLength(0);
    expect(removeMatchesForUser(list, "u2")).toHaveLength(0);
  });
  it("tracks unread counts", () => {
    const payload = { items: [notification("a", true)], unreadCount: 0 };
    const added = applyNotification(payload, notification("b"));
    expect(added.unreadCount).toBe(1);
    expect(applyNotification(added, notification("b"))).toBe(added);
    const read = markNotificationReadInPayload(added, "b");
    expect(read.unreadCount).toBe(0);
    expect(markNotificationReadInPayload(read, "b")).toBe(read);
    expect(markAllReadInPayload(added).items.every((n) => n.isRead)).toBe(true);
    expect(markAllReadInPayload(added).unreadCount).toBe(0);
  });
});
