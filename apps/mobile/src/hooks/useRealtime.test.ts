import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import type { MessagePage } from "../api/platform";
import type { ChatMessage } from "../api/types";
import { queryKeys } from "../lib/queryClient";
import { appendMessageToCache, handleRealtimeEvent, markSentMessagesRead } from "./useRealtime";

const message = (id: number, senderId = "other"): ChatMessage => ({
  id,
  conversationId: "c1",
  senderId,
  body: `m${id}`,
  createdAt: new Date(2026, 0, 1, 10, id).toISOString(),
  read: false
});

function seed(client: QueryClient, messages: ChatMessage[]) {
  const data: InfiniteData<MessagePage, number | null> = { pages: [{ messages, nextBefore: null }], pageParams: [null] };
  client.setQueryData(queryKeys.messages("c1"), data);
}

const cached = (client: QueryClient) => client.getQueryData<InfiniteData<MessagePage>>(queryKeys.messages("c1"))?.pages[0].messages;

const clients: QueryClient[] = [];
function newClient(): QueryClient {
  const client = new QueryClient();
  clients.push(client);
  return client;
}

afterEach(() => {
  clients.splice(0).forEach((client) => client.clear());
});

describe("realtime cache updates", () => {
  it("appends a pushed message once", () => {
    const client = newClient();
    seed(client, [message(1)]);
    appendMessageToCache(client, message(2));
    appendMessageToCache(client, message(2));
    expect(cached(client)?.map((m) => m.id)).toEqual([1, 2]);
  });

  it("ignores messages for conversations that are not loaded", () => {
    const client = newClient();
    appendMessageToCache(client, message(1));
    expect(client.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });

  it("marks sent messages as read up to the read time", () => {
    const client = newClient();
    seed(client, [message(1, "me"), message(2, "me"), message(3, "me")]);
    markSentMessagesRead(client, "c1", new Date(2026, 0, 1, 10, 2).toISOString());
    expect(cached(client)?.map((m) => m.read)).toEqual([true, true, false]);
  });

  it("refreshes lists on message and match events", () => {
    const client = newClient();
    const invalidate = jest.spyOn(client, "invalidateQueries");
    handleRealtimeEvent(client, { type: "message", data: message(5) }, "me");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.conversations });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.notifications });

    invalidate.mockClear();
    handleRealtimeEvent(client, { type: "message", data: message(6, "me") }, "me");
    expect(invalidate).not.toHaveBeenCalledWith({ queryKey: queryKeys.notifications });

    invalidate.mockClear();
    handleRealtimeEvent(client, { type: "unmatch" }, "me");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.matches });
  });
});
