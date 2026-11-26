import type { InfiniteData } from "@tanstack/react-query";
import type { MessagesPage } from "../api/types";
import { queryClient, queryKeys } from "./queryClient";
import { applyRealtimeEvent } from "./useRealtime";

jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn(),
  setItemAsync: jest.fn(),
  deleteItemAsync: jest.fn()
}));

const conv = "c1";
const page = (messages: MessagesPage["messages"], otherLastReadAt: string | null = null): InfiniteData<MessagesPage> => ({
  pages: [{ messages, nextCursor: null, otherLastReadAt }],
  pageParams: [null]
});

describe("applyRealtimeEvent", () => {
  afterEach(() => queryClient.clear());

  it("prepends incoming messages once", () => {
    queryClient.setQueryData(
      queryKeys.messages(conv),
      page([{ id: "m1", conversationId: conv, senderId: "u2", body: "old", createdAt: "2026-01-01T10:00:00Z" }])
    );
    const message = { id: "m2", conversationId: conv, senderId: "u2", body: "new", createdAt: "2026-01-01T10:01:00Z" };
    applyRealtimeEvent({ type: "message.created", data: message }, "u1");
    applyRealtimeEvent({ type: "message.created", data: message }, "u1");
    const data = queryClient.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conv));
    expect(data?.pages[0].messages.map((m) => m.id)).toEqual(["m2", "m1"]);
  });

  it("records read receipts from the other member only", () => {
    queryClient.setQueryData(queryKeys.messages(conv), page([]));
    applyRealtimeEvent(
      { type: "conversation.read", data: { conversationId: conv, userId: "u1", lastReadAt: "2026-01-01T09:00:00Z" } },
      "u1"
    );
    expect(queryClient.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conv))?.pages[0].otherLastReadAt).toBeNull();
    applyRealtimeEvent(
      { type: "conversation.read", data: { conversationId: conv, userId: "u2", lastReadAt: "2026-01-01T11:00:00Z" } },
      "u1"
    );
    expect(queryClient.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conv))?.pages[0].otherLastReadAt).toBe(
      "2026-01-01T11:00:00Z"
    );
  });
});
