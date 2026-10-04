import { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { applyRealtimeEvent, type RealtimeContext } from "../realtimeCache";
import type { AppNotification, MatchSummary, Message } from "../types";
import type { CursorData } from "../../lib/pagination";

const ME = "me";
const THEM = "them";

const message = (over: Partial<Message> = {}): Message => ({
  id: "m1",
  conversationId: "c1",
  senderId: THEM,
  body: "hi",
  createdAt: "2026-01-01T10:00:00Z",
  readAt: null,
  ...over
});

const summary = (over: Partial<MatchSummary> = {}): MatchSummary => ({
  id: "match1",
  conversationId: "c1",
  matchedAt: "2026-01-01T09:00:00Z",
  user: { userId: THEM, displayName: "Sam", age: 30, photo: null },
  lastMessage: null,
  unreadCount: 0,
  updatedAt: "2026-01-01T09:00:00Z",
  ...over
});

function pages<K extends string, T>(key: K, items: T[]): CursorData<K, T> {
  return { pages: [{ [key]: items, nextCursor: null } as never], pageParams: [null] };
}

function setup(active: string | null = null) {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
  const ctx: RealtimeContext = { myUserId: ME, getActiveConversationId: () => active };
  return { client, ctx };
}

describe("applyRealtimeEvent", () => {
  it("message.new: prepends to messages, updates the conversation and unread counts", () => {
    const { client, ctx } = setup();
    client.setQueryData(queryKeys.messages("c1"), pages("messages", [message({ id: "old" })]));
    client.setQueryData(
      queryKeys.conversations,
      pages("conversations", [summary(), summary({ id: "match2", conversationId: "c2" })])
    );
    client.setQueryData(queryKeys.conversationsUnread, 0);

    applyRealtimeEvent(client, { type: "message.new", data: message({ id: "m2" }) }, ctx);

    const msgs = client.getQueryData<CursorData<"messages", Message>>(queryKeys.messages("c1"));
    expect(msgs?.pages[0].messages.map((m) => m.id)).toEqual(["m2", "old"]);
    const convs = client.getQueryData<CursorData<"conversations", MatchSummary>>(queryKeys.conversations);
    expect(convs?.pages[0].conversations[0]).toMatchObject({
      conversationId: "c1",
      unreadCount: 1,
      lastMessage: { id: "m2" }
    });
    expect(client.getQueryData(queryKeys.conversationsUnread)).toBe(1);
  });

  it("message.new is idempotent and does not add unread for the open chat or my own messages", () => {
    const { client, ctx } = setup("c1");
    client.setQueryData(queryKeys.messages("c1"), pages("messages", []));
    client.setQueryData(queryKeys.conversations, pages("conversations", [summary()]));
    client.setQueryData(queryKeys.conversationsUnread, 0);
    const event = { type: "message.new", data: message({ id: "dup" }) } as const;
    applyRealtimeEvent(client, event, ctx);
    applyRealtimeEvent(client, event, ctx);
    const msgs = client.getQueryData<CursorData<"messages", Message>>(queryKeys.messages("c1"));
    expect(msgs?.pages[0].messages).toHaveLength(1);
    expect(client.getQueryData(queryKeys.conversationsUnread)).toBe(0);

    applyRealtimeEvent(
      client,
      { type: "message.new", data: message({ id: "mine", senderId: ME }) },
      setup("other").ctx
    );
    expect(client.getQueryData(queryKeys.conversationsUnread)).toBe(0);
  });

  it("message.read marks only my messages as read", () => {
    const { client, ctx } = setup();
    client.setQueryData(
      queryKeys.messages("c1"),
      pages("messages", [message({ id: "a", senderId: ME }), message({ id: "b", senderId: THEM })])
    );
    applyRealtimeEvent(
      client,
      { type: "message.read", data: { conversationId: "c1", readAt: "2026-01-01T11:00:00Z" } },
      ctx
    );
    const msgs = client.getQueryData<CursorData<"messages", Message>>(queryKeys.messages("c1"))?.pages[0].messages;
    expect(msgs?.find((m) => m.id === "a")?.readAt).toBe("2026-01-01T11:00:00Z");
    expect(msgs?.find((m) => m.id === "b")?.readAt).toBeNull();
  });

  it("match.new prepends to matches; match.removed drops it everywhere", () => {
    const { client, ctx } = setup();
    client.setQueryData(queryKeys.matches, pages("matches", [summary({ id: "other", conversationId: "cx" })]));
    client.setQueryData(queryKeys.conversations, pages("conversations", [summary()]));
    client.setQueryData(queryKeys.messages("c1"), pages("messages", [message()]));

    applyRealtimeEvent(client, { type: "match.new", data: summary() }, ctx);
    expect(
      client.getQueryData<CursorData<"matches", MatchSummary>>(queryKeys.matches)?.pages[0].matches.map((m) => m.id)
    ).toEqual(["match1", "other"]);

    applyRealtimeEvent(client, { type: "match.removed", data: { matchId: "match1", conversationId: "c1" } }, ctx);
    expect(
      client.getQueryData<CursorData<"matches", MatchSummary>>(queryKeys.matches)?.pages[0].matches.map((m) => m.id)
    ).toEqual(["other"]);
    expect(
      client.getQueryData<CursorData<"conversations", MatchSummary>>(queryKeys.conversations)?.pages[0].conversations
    ).toHaveLength(0);
    expect(client.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });

  it("notification.new prepends and bumps the unread badge", () => {
    const { client, ctx } = setup();
    const n: AppNotification = {
      id: "n1",
      type: "match",
      title: "t",
      body: "b",
      data: {},
      readAt: null,
      createdAt: "2026-01-01T00:00:00Z"
    };
    client.setQueryData(queryKeys.notifications, pages("notifications", []));
    client.setQueryData(queryKeys.notificationsUnread, 2);
    applyRealtimeEvent(client, { type: "notification.new", data: n }, ctx);
    expect(
      client.getQueryData<CursorData<"notifications", AppNotification>>(queryKeys.notifications)?.pages[0]
        .notifications[0].id
    ).toBe("n1");
    expect(client.getQueryData(queryKeys.notificationsUnread)).toBe(3);
  });
});
