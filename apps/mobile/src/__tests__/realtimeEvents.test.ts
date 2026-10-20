import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { queryKeys } from "../api/keys";
import type { ConversationsPage, Message, MessagesPage } from "../api/types";
import { applyRealtimeEvent, removeMessage, replaceMessage } from "../realtime/events";

const msg = (id: string, senderId: string, conversationId = "c1"): Message => ({
  id,
  conversationId,
  senderId,
  body: id,
  createdAt: "2026-01-01T10:00:00Z"
});

const messagesData = (messages: Message[]): InfiniteData<MessagesPage, string | null> => ({
  pages: [{ messages, otherLastReadAt: null, nextCursor: null }],
  pageParams: [null]
});

const conversation = (id: string, unread = 0) => ({
  id,
  user: { userId: `u-${id}`, firstName: id, photoUrl: null },
  lastMessage: null,
  unreadCount: unread,
  updatedAt: "2026-01-01T00:00:00Z"
});

const conversationsData = (): InfiniteData<ConversationsPage, string | null> => ({
  pages: [{ conversations: [conversation("c2"), conversation("c1")], nextCursor: null }],
  pageParams: [null]
});

describe("applyRealtimeEvent", () => {
  const ctx = { userId: "me", activeConversationId: null };

  it("adds incoming messages once and bumps unread + order", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    qc.setQueryData(queryKeys.messages("c1"), messagesData([msg("m0", "other")]));
    qc.setQueryData(queryKeys.conversations, conversationsData());

    const event = { type: "message.new", message: msg("m1", "other") } as const;
    applyRealtimeEvent(qc, event, ctx);
    applyRealtimeEvent(qc, event, ctx);

    const messages = qc.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages("c1"));
    expect(messages?.pages[0].messages.map((m) => m.id)).toEqual(["m1", "m0"]);
    const convs = qc.getQueryData<InfiniteData<ConversationsPage>>(queryKeys.conversations);
    expect(convs?.pages[0].conversations.map((c) => c.id)).toEqual(["c1", "c2"]);
    expect(convs?.pages[0].conversations[0].unreadCount).toBe(2);
  });

  it("does not count unread for the open conversation or own messages", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    qc.setQueryData(queryKeys.conversations, conversationsData());
    applyRealtimeEvent(
      qc,
      { type: "message.new", message: msg("m1", "other") },
      { ...ctx, activeConversationId: "c1" }
    );
    applyRealtimeEvent(qc, { type: "message.new", message: msg("m2", "me") }, ctx);
    const convs = qc.getQueryData<InfiniteData<ConversationsPage>>(queryKeys.conversations);
    expect(convs?.pages[0].conversations.find((c) => c.id === "c1")?.unreadCount).toBe(0);
  });

  it("records the other participant's read receipt only", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    qc.setQueryData(queryKeys.messages("c1"), messagesData([msg("m1", "me")]));
    applyRealtimeEvent(
      qc,
      { type: "message.read", conversationId: "c1", userId: "me", readAt: "2026-01-02T00:00:00Z" },
      ctx
    );
    expect(
      qc.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages("c1"))?.pages[0].otherLastReadAt
    ).toBeNull();
    applyRealtimeEvent(
      qc,
      { type: "message.read", conversationId: "c1", userId: "other", readAt: "2026-01-02T00:00:00Z" },
      ctx
    );
    expect(
      qc.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages("c1"))?.pages[0].otherLastReadAt
    ).toBe("2026-01-02T00:00:00Z");
  });

  it("invalidates matches on match.new", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    const spy = jest.spyOn(qc, "invalidateQueries");
    applyRealtimeEvent(
      qc,
      {
        type: "match.new",
        match: {
          id: "m",
          user: { userId: "u", firstName: "A", photoUrl: null },
          conversationId: "c",
          createdAt: ""
        }
      },
      ctx
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.matches });
  });
});

describe("match.removed", () => {
  it("evicts the thread and refreshes matches and conversations", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    qc.setQueryData(queryKeys.messages("c1"), { pages: [], pageParams: [] });
    const spy = jest.spyOn(qc, "invalidateQueries");
    const ctx = { userId: "me", activeConversationId: null };
    applyRealtimeEvent(qc, { type: "match.removed", matchId: "m1", conversationId: "c1" }, ctx);
    expect(qc.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.matches });
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.conversations });
  });
});

describe("optimistic message helpers", () => {
  it("replaces a temp message by the server copy, deduping websocket echoes", () => {
    const temp = msg("tmp-1", "me");
    const real = msg("m9", "me");
    const replaced = replaceMessage(messagesData([temp]), "tmp-1", real);
    expect(replaced?.pages[0].messages.map((m) => m.id)).toEqual(["m9"]);
    const echoFirst = replaceMessage(messagesData([real, temp]), "tmp-1", real);
    expect(echoFirst?.pages[0].messages.map((m) => m.id)).toEqual(["m9"]);
    expect(removeMessage(messagesData([temp]), "tmp-1")?.pages[0].messages).toEqual([]);
  });
});
