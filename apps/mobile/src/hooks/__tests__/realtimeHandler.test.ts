import { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../api/queryKeys";
import type { MatchSummary, Message } from "../../api/types";
import type { MatchPages, MessagePages } from "../../domain/realtimeCache";
import { applyRealtimeEvent } from "../realtimeHandler";

jest.mock("../../api/matches", () => ({ markConversationRead: jest.fn(() => Promise.resolve()) }));

const match: MatchSummary = {
  matchId: "m1",
  conversationId: "c1",
  createdAt: "2026-01-01T00:00:00Z",
  user: { userId: "u1", firstName: "Ana", age: 30, photo: null },
  lastMessage: null,
  unreadCount: 0
};

const incoming: Message = {
  id: "msg1",
  conversationId: "c1",
  senderId: "u1",
  body: "hello",
  createdAt: "2026-01-01T10:00:00Z",
  readAt: null
};

const clients: QueryClient[] = [];

afterEach(() => {
  clients.splice(0).forEach((client) => client.clear());
});

function setup() {
  const client = new QueryClient();
  clients.push(client);
  client.setQueryData<MatchPages>(queryKeys.matches, {
    pages: [{ items: [match], nextCursor: null }],
    pageParams: [null]
  });
  client.setQueryData<MessagePages>(queryKeys.messages("c1"), {
    pages: [{ items: [], nextCursor: null }],
    pageParams: [null]
  });
  const notify = jest.fn();
  return { client, notify, context: { myUserId: "me", openConversationId: null, notify } };
}

describe("applyRealtimeEvent", () => {
  it("message.new updates both the thread and the conversation list", () => {
    const { client, context } = setup();
    applyRealtimeEvent(client, { type: "message.new", data: incoming }, context);
    expect(client.getQueryData<MessagePages>(queryKeys.messages("c1"))?.pages[0].items).toHaveLength(1);
    const matches = client.getQueryData<MatchPages>(queryKeys.matches);
    expect(matches?.pages[0].items[0].unreadCount).toBe(1);
  });

  it("message.new in the open conversation is not counted as unread", () => {
    const { client, context } = setup();
    applyRealtimeEvent(client, { type: "message.new", data: incoming }, { ...context, openConversationId: "c1" });
    expect(client.getQueryData<MatchPages>(queryKeys.matches)?.pages[0].items[0].unreadCount).toBe(0);
  });

  it("messages.read from the other person marks my messages read", () => {
    const { client, context } = setup();
    client.setQueryData<MessagePages>(queryKeys.messages("c1"), {
      pages: [{ items: [{ ...incoming, id: "mine", senderId: "me" }], nextCursor: null }],
      pageParams: [null]
    });
    applyRealtimeEvent(client, { type: "messages.read", data: { conversationId: "c1", readerId: "u1" } }, context);
    expect(client.getQueryData<MessagePages>(queryKeys.messages("c1"))?.pages[0].items[0].readAt).not.toBeNull();
  });

  it("match.new adds the match and notifies once", () => {
    const { client, context, notify } = setup();
    const other = { ...match, matchId: "m2", conversationId: "c2", user: { ...match.user, firstName: "Bea" } };
    applyRealtimeEvent(client, { type: "match.new", data: other }, context);
    applyRealtimeEvent(client, { type: "match.new", data: other }, context);
    expect(client.getQueryData<MatchPages>(queryKeys.matches)?.pages[0].items).toHaveLength(2);
    expect(notify).toHaveBeenCalledTimes(1);
    expect(notify).toHaveBeenCalledWith("It's a match with Bea!");
  });

  it("match.removed drops the match and its cached thread", () => {
    const { client, context } = setup();
    applyRealtimeEvent(client, { type: "match.removed", data: { matchId: "m1", conversationId: "c1" } }, context);
    expect(client.getQueryData<MatchPages>(queryKeys.matches)?.pages[0].items).toEqual([]);
    expect(client.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });

  it("notification.new prepends and raises the unread count", () => {
    const { client, context } = setup();
    client.setQueryData(queryKeys.notifications, {
      pages: [{ items: [], nextCursor: null, unreadCount: 0 }],
      pageParams: [null]
    });
    applyRealtimeEvent(
      client,
      {
        type: "notification.new",
        data: {
          id: "n1",
          type: "message",
          title: "t",
          body: "b",
          data: {},
          readAt: null,
          createdAt: "2026-01-01T00:00:00Z"
        }
      },
      context
    );
    const data = client.getQueryData<{ pages: { items: unknown[]; unreadCount: number }[] }>(queryKeys.notifications);
    expect(data?.pages[0].unreadCount).toBe(1);
    expect(data?.pages[0].items).toHaveLength(1);
  });
});
