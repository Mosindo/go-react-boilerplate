import { QueryClient } from "@tanstack/react-query";
import type {
  MatchesPage,
  MatchSummary,
  Message,
  MessagesPage,
  NotificationItem,
  NotificationsPage
} from "../src/api/types";
import {
  appendMessage,
  applyMatchNew,
  applyMatchRemoved,
  applyMessageNewToMatches,
  clearUnread,
  flattenMessages,
  markAllNotificationsReadInCache,
  markMessagesRead,
  markNotificationReadInCache,
  notificationUnreadCount,
  prependNotification,
  totalUnread,
  type Paged
} from "../src/realtime/cache";
import { applyServerEvent, parseServerEvent } from "../src/realtime/events";
import { queryKeys } from "../src/session/queryKeys";
import { activeChat } from "../src/realtime/activeChat";
import { buildChatEntries, lastOwnMessageId } from "../src/features/outbox";

const paged = <P>(...pages: P[]): Paged<P> => ({ pages, pageParams: pages.map(() => null) });

const match = (id: string, extra: Partial<MatchSummary> = {}): MatchSummary => ({
  id,
  createdAt: "2025-01-01T00:00:00Z",
  lastActivityAt: "2025-01-01T00:00:00Z",
  unreadCount: 0,
  user: { userId: `u-${id}`, firstName: id, thumbUrl: null },
  lastMessage: null,
  ...extra
});

const message = (id: string, extra: Partial<Message> = {}): Message => ({
  id,
  matchId: "m1",
  senderId: "other",
  content: `msg ${id}`,
  createdAt: "2025-01-02T10:00:00Z",
  readAt: null,
  ...extra
});

const notification = (id: string, readAt: string | null = null): NotificationItem => ({
  id,
  type: "message",
  title: "t",
  body: "b",
  data: { matchId: "m1" },
  readAt,
  createdAt: "2025-01-02T10:00:00Z"
});

describe("matches cache", () => {
  const data = paged<MatchesPage>(
    { matches: [match("m1"), match("m2", { unreadCount: 2 })], nextCursor: "c" },
    { matches: [match("m3", { unreadCount: 1 })], nextCursor: null }
  );

  it("message.new bumps unread, updates last message and moves the match to the top", () => {
    const next = applyMessageNewToMatches(data, message("x", { matchId: "m3" }), "me", null);
    expect(next).toBeTruthy();
    const first = next!.pages[0].matches[0];
    expect(first.id).toBe("m3");
    expect(first.unreadCount).toBe(2);
    expect(first.lastMessage?.content).toBe("msg x");
    expect(next!.pages[1].matches.find((m) => m.id === "m3")).toBeUndefined();
  });

  it("does not raise unread for my own messages or for the open conversation", () => {
    const mine = applyMessageNewToMatches(
      data,
      message("x", { matchId: "m2", senderId: "me" }),
      "me",
      null
    );
    expect(mine!.pages[0].matches[0].unreadCount).toBe(2);
    const open = applyMessageNewToMatches(data, message("y", { matchId: "m2" }), "me", "m2");
    expect(open!.pages[0].matches[0].unreadCount).toBe(2);
  });

  it("returns null for unknown matches (refetch) and undefined without cache", () => {
    expect(applyMessageNewToMatches(data, message("x", { matchId: "zzz" }), "me", null)).toBeNull();
    expect(applyMessageNewToMatches(undefined, message("x"), "me", null)).toBeUndefined();
  });

  it("match.new prepends, match.removed deletes, clearUnread zeroes", () => {
    const added = applyMatchNew(data, match("m9"))!;
    expect(added.pages[0].matches[0].id).toBe("m9");
    expect(
      applyMatchNew(added, match("m9", { unreadCount: 5 }))!.pages[0].matches[0].unreadCount
    ).toBe(5);
    expect(applyMatchNew(added, match("m9"))!.pages.flatMap((p) => p.matches)).toHaveLength(4);
    const removed = applyMatchRemoved(data, "m2")!;
    expect(removed.pages.flatMap((p) => p.matches).map((m) => m.id)).toEqual(["m1", "m3"]);
    expect(totalUnread(data)).toBe(3);
    expect(totalUnread(clearUnread(data, "m2"))).toBe(1);
    expect(totalUnread(undefined)).toBe(0);
  });
});

describe("messages cache", () => {
  const data = paged<MessagesPage>({ messages: [message("2"), message("1")], nextCursor: null });

  it("appends at the top and ignores duplicates", () => {
    const next = appendMessage(data, message("3"))!;
    expect(flattenMessages(next).map((m) => m.id)).toEqual(["3", "2", "1"]);
    expect(appendMessage(next, message("3"))).toBe(next);
    expect(appendMessage(undefined, message("3"))).toBeUndefined();
  });

  it("message.read marks the other side's unread messages as read", () => {
    const mixed = paged<MessagesPage>({
      messages: [
        message("3", { senderId: "me" }),
        message("2", { senderId: "other" }),
        message("1", { senderId: "me", readAt: "2025-01-01T00:00:00Z" })
      ],
      nextCursor: null
    });
    const next = markMessagesRead(mixed, "other", "2025-01-03T00:00:00Z")!;
    const byId = Object.fromEntries(flattenMessages(next).map((m) => [m.id, m.readAt]));
    expect(byId["3"]).toBe("2025-01-03T00:00:00Z");
    expect(byId["2"]).toBeNull();
    expect(byId["1"]).toBe("2025-01-01T00:00:00Z");
  });
});

describe("notifications cache", () => {
  const data = paged<NotificationsPage>({
    notifications: [notification("n1"), notification("n2", "2025-01-01T00:00:00Z")],
    unreadCount: 1,
    nextCursor: null
  });

  it("prepends and counts unread without duplicating", () => {
    const next = prependNotification(data, notification("n3"))!;
    expect(next.pages[0].notifications[0].id).toBe("n3");
    expect(notificationUnreadCount(next)).toBe(2);
    expect(prependNotification(next, notification("n3"))).toBe(next);
  });

  it("marks one or all as read", () => {
    const one = markNotificationReadInCache(data, "n1", "2025-02-01T00:00:00Z")!;
    expect(notificationUnreadCount(one)).toBe(0);
    expect(markNotificationReadInCache(one, "n1", "x")).toBe(one);
    const all = markAllNotificationsReadInCache(
      prependNotification(data, notification("n4")),
      "2025-02-01T00:00:00Z"
    )!;
    expect(notificationUnreadCount(all)).toBe(0);
    expect(all.pages[0].notifications.every((n) => n.readAt)).toBe(true);
  });
});

describe("parseServerEvent", () => {
  it("accepts known events and rejects control or malformed frames", () => {
    expect(
      parseServerEvent(JSON.stringify({ type: "match.removed", data: { matchId: "m1" } }))
    ).toEqual({
      type: "match.removed",
      data: { matchId: "m1" }
    });
    expect(parseServerEvent(JSON.stringify({ type: "ready" }))).toBeNull();
    expect(parseServerEvent(JSON.stringify({ type: "unknown", data: {} }))).toBeNull();
    expect(parseServerEvent("not json")).toBeNull();
    expect(parseServerEvent(42)).toBeNull();
  });
});

describe("applyServerEvent", () => {
  const clients: QueryClient[] = [];
  afterEach(() => {
    clients.splice(0).forEach((client) => client.clear());
  });

  function setup() {
    const client = new QueryClient();
    clients.push(client);
    client.setQueryData(
      queryKeys.matches,
      paged<MatchesPage>({ matches: [match("m1")], nextCursor: null })
    );
    client.setQueryData(
      queryKeys.messages("m1"),
      paged<MessagesPage>({ messages: [], nextCursor: null })
    );
    client.setQueryData(
      queryKeys.notifications,
      paged<NotificationsPage>({ notifications: [], unreadCount: 0, nextCursor: null })
    );
    return client;
  }
  const ctx = { myUserId: "me", activeMatchId: null };

  it("routes message.new into both caches", () => {
    const client = setup();
    applyServerEvent(client, { type: "message.new", data: message("a") }, ctx);
    expect(flattenMessages(client.getQueryData(queryKeys.messages("m1")))).toHaveLength(1);
    expect(totalUnread(client.getQueryData(queryKeys.matches))).toBe(1);
  });

  it("routes message.read, match.new, match.removed and notification.new", () => {
    const client = setup();
    applyServerEvent(client, { type: "message.new", data: message("a", { senderId: "me" }) }, ctx);
    applyServerEvent(
      client,
      {
        type: "message.read",
        data: { matchId: "m1", readerId: "other", readAt: "2025-01-05T00:00:00Z" }
      },
      ctx
    );
    expect(flattenMessages(client.getQueryData(queryKeys.messages("m1")))[0].readAt).toBe(
      "2025-01-05T00:00:00Z"
    );

    applyServerEvent(client, { type: "match.new", data: match("m2") }, ctx);
    expect(client.getQueryData<Paged<MatchesPage>>(queryKeys.matches)!.pages[0].matches[0].id).toBe(
      "m2"
    );

    const removedListener = jest.fn();
    const stop = activeChat.onRemoved(removedListener);
    applyServerEvent(client, { type: "match.removed", data: { matchId: "m1" } }, ctx);
    stop();
    expect(removedListener).toHaveBeenCalledWith("m1");
    expect(client.getQueryData(queryKeys.messages("m1"))).toBeUndefined();
    expect(
      client.getQueryData<Paged<MatchesPage>>(queryKeys.matches)!.pages[0].matches.map((m) => m.id)
    ).toEqual(["m2"]);

    applyServerEvent(client, { type: "notification.new", data: notification("n1") }, ctx);
    expect(notificationUnreadCount(client.getQueryData(queryKeys.notifications))).toBe(1);
  });

  it("invalidates the matches list when a message arrives for an unknown match", () => {
    const client = setup();
    const spy = jest.spyOn(client, "invalidateQueries");
    applyServerEvent(
      client,
      { type: "message.new", data: message("a", { matchId: "unknown" }) },
      ctx
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.matches });
  });
});

describe("chat outbox", () => {
  it("puts pending messages first (newest first) and finds my last message", () => {
    const entries = buildChatEntries(
      [message("2", { senderId: "me" }), message("1")],
      [
        { localId: "l1", content: "a", createdAt: "x", status: "sending" },
        { localId: "l2", content: "b", createdAt: "y", status: "failed" }
      ]
    );
    expect(entries.map((e) => (e.kind === "pending" ? e.pending.localId : e.message.id))).toEqual([
      "l2",
      "l1",
      "2",
      "1"
    ]);
    expect(lastOwnMessageId([message("2", { senderId: "me" }), message("1")], "me")).toBe("2");
    expect(lastOwnMessageId([message("1")], "me")).toBeNull();
  });
});
