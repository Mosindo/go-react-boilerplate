import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { queryKeys } from "../../api/queryClient";
import type { Message } from "../../api/types";
import { applyRealtimeEvent, setActiveConversation, toSocketUrl, type MessagesPage } from "../useRealtime";

jest.mock("../../shared/feedback/store", () => ({ showToast: jest.fn(), showGlobalError: jest.fn() }));
// eslint-disable-next-line import/first
import { showToast } from "../../shared/feedback/store";

const newClient = () => new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });

const message = (id: string, senderId: string, readAt: string | null = null): Message => ({
  id,
  conversationId: "c1",
  senderId,
  body: `body ${id}`,
  createdAt: "2026-06-15T10:00:00Z",
  readAt
});

function seed(client: QueryClient, messages: Message[]) {
  const data: InfiniteData<MessagesPage> = { pages: [{ messages, hasMore: false }], pageParams: [undefined] };
  client.setQueryData(queryKeys.messages("c1"), data);
}

const read = (client: QueryClient) => client.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages("c1"))?.pages[0]?.messages ?? [];

describe("toSocketUrl", () => {
  it("switches scheme and encodes the ticket", () => {
    expect(toSocketUrl("http://10.0.0.2:18080", "a.b/c")).toBe("ws://10.0.0.2:18080/ws?ticket=a.b%2Fc");
    expect(toSocketUrl("https://api.example.com", "t")).toBe("wss://api.example.com/ws?ticket=t");
  });
});

describe("applyRealtimeEvent", () => {
  beforeEach(() => {
    (showToast as jest.Mock).mockClear();
    setActiveConversation(null);
  });

  it("appends an incoming message once (REST echo + socket do not duplicate)", () => {
    const client = newClient();
    seed(client, [message("m1", "them")]);
    const event = { type: "message" as const, data: message("m2", "them") };
    applyRealtimeEvent(client, event, "me");
    applyRealtimeEvent(client, event, "me");
    expect(read(client).map((m) => m.id)).toEqual(["m1", "m2"]);
  });

  it("toasts for messages from others unless that conversation is open", () => {
    const client = newClient();
    seed(client, []);
    applyRealtimeEvent(client, { type: "message", data: message("m1", "them") }, "me");
    expect(showToast).toHaveBeenCalledTimes(1);

    setActiveConversation("c1");
    applyRealtimeEvent(client, { type: "message", data: message("m2", "them") }, "me");
    applyRealtimeEvent(client, { type: "message", data: message("m3", "me") }, "me");
    expect(showToast).toHaveBeenCalledTimes(1);
  });

  it("marks only my messages as read on a read receipt", () => {
    const client = newClient();
    seed(client, [message("m1", "me"), message("m2", "them")]);
    applyRealtimeEvent(client, { type: "read", data: { conversationId: "c1", readerId: "them", readAt: "2026-06-15T11:00:00Z" } }, "me");
    const [mine, theirs] = read(client);
    expect(mine?.readAt).toBe("2026-06-15T11:00:00Z");
    expect(theirs?.readAt).toBeNull();
  });

  it("ignores messages for conversations that are not cached", () => {
    const client = newClient();
    applyRealtimeEvent(client, { type: "message", data: message("m1", "them") }, "me");
    expect(client.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });
});
