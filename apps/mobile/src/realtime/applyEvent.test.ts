import { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../api";
import { conversation, matchItem, message, notification, notificationPages, pages } from "../test-utils";
import { applyRealtimeEvent, type EventContext } from "./applyEvent";
import type { NotificationPages, Paged } from "./cache";
import type { ConversationItem, MatchItem, Message } from "../api";

function setup(overrides: Partial<EventContext> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
  const context: EventContext = {
    currentUserId: "me",
    activeConversationId: null,
    onMatch: jest.fn(),
    markRead: jest.fn(),
    ...overrides
  };
  queryClient.setQueryData(queryKeys.messages("c1"), pages([message({ id: "m0" })]));
  queryClient.setQueryData(queryKeys.conversations, pages([conversation({ lastMessage: message({ id: "m0" }) })]));
  queryClient.setQueryData(queryKeys.unreadCount, 0);
  return { queryClient, context };
}

describe("applyRealtimeEvent", () => {
  it("appends an incoming message to the thread, the list and the unread counter", () => {
    const { queryClient, context } = setup();
    applyRealtimeEvent(queryClient, { type: "message", data: message({ id: "m1" }) }, context);

    expect(queryClient.getQueryData<Paged<Message>>(queryKeys.messages("c1"))?.pages[0]?.items[0]?.id).toBe("m1");
    expect(
      queryClient.getQueryData<Paged<ConversationItem>>(queryKeys.conversations)?.pages[0]?.items[0]?.unreadCount
    ).toBe(1);
    expect(queryClient.getQueryData(queryKeys.unreadCount)).toBe(1);
    expect(context.markRead).not.toHaveBeenCalled();
  });

  it("marks the thread read instead of counting unread when it is open", () => {
    const { queryClient, context } = setup({ activeConversationId: "c1" });
    applyRealtimeEvent(queryClient, { type: "message", data: message({ id: "m1" }) }, context);
    expect(context.markRead).toHaveBeenCalledWith("c1");
    expect(queryClient.getQueryData(queryKeys.unreadCount)).toBe(0);
  });

  it("ignores the echo of my own message already present", () => {
    const { queryClient, context } = setup();
    const mine = message({ id: "m5", senderId: "me" });
    applyRealtimeEvent(queryClient, { type: "message", data: mine }, context);
    applyRealtimeEvent(queryClient, { type: "message", data: mine }, context);
    expect(queryClient.getQueryData<Paged<Message>>(queryKeys.messages("c1"))?.pages[0]?.items).toHaveLength(2);
    expect(queryClient.getQueryData(queryKeys.unreadCount)).toBe(0);
  });

  it("applies read receipts", () => {
    const { queryClient, context } = setup();
    queryClient.setQueryData(queryKeys.messages("c1"), pages([message({ id: "mine", senderId: "me" })]));
    applyRealtimeEvent(
      queryClient,
      { type: "read", data: { conversationId: "c1", readerId: "other", readAt: "2026-03-10T13:00:00.000Z" } },
      context
    );
    expect(queryClient.getQueryData<Paged<Message>>(queryKeys.messages("c1"))?.pages[0]?.items[0]?.readAt).toBe(
      "2026-03-10T13:00:00.000Z"
    );
  });

  it("announces a match and caches it", () => {
    const { queryClient, context } = setup();
    queryClient.setQueryData(queryKeys.matches, pages<MatchItem>([]));
    const match = matchItem({ matchId: "new" });
    applyRealtimeEvent(queryClient, { type: "match", data: match }, context);
    expect(context.onMatch).toHaveBeenCalledWith(match);
    expect(queryClient.getQueryData<Paged<MatchItem>>(queryKeys.matches)?.pages[0]?.items[0]?.matchId).toBe("new");
  });

  it("removes a conversation and its messages", () => {
    const { queryClient, context } = setup();
    applyRealtimeEvent(queryClient, { type: "conversation_removed", data: { conversationId: "c1" } }, context);
    expect(queryClient.getQueryData<Paged<ConversationItem>>(queryKeys.conversations)?.pages[0]?.items).toEqual([]);
    expect(queryClient.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });

  it("feeds notifications", () => {
    const { queryClient, context } = setup();
    queryClient.setQueryData(queryKeys.notifications, notificationPages([], 0));
    applyRealtimeEvent(queryClient, { type: "notification", data: notification() }, context);
    const data = queryClient.getQueryData<NotificationPages>(queryKeys.notifications);
    expect(data?.pages[0]?.items).toHaveLength(1);
    expect(data?.pages[0]?.unreadCount).toBe(1);
  });
});
