import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { keys } from "../api/keys";
import type { Message, MessagesPage } from "../api/types";
import { applyRealtimeEvent, parseEvent, reconnectDelay } from "./events";

const msg = (id: string): Message => ({ id, conversationId: "c1", senderId: "u2", body: id, createdAt: "2026-03-15T10:00:00Z" });
const seed = (client: QueryClient) =>
  client.setQueryData<InfiniteData<MessagesPage>>(keys.messages("c1"), {
    pages: [{ messages: [msg("m1")], otherLastReadAt: "2026-03-15T09:00:00Z" }],
    pageParams: [undefined]
  });

describe("parseEvent", () => {
  it("accepts known events and ignores the rest", () => {
    expect(parseEvent('{"type":"match.removed","data":{"matchId":"x"}}')?.type).toBe("match.removed");
    expect(parseEvent('{"type":"unknown","data":{}}')).toBeNull();
    expect(parseEvent("not json")).toBeNull();
  });
});

describe("applyRealtimeEvent", () => {
  it("prepends a new message once", () => {
    const client = new QueryClient();
    seed(client);
    const event = { type: "message.new", data: msg("m2") } as const;
    applyRealtimeEvent(client, event);
    applyRealtimeEvent(client, event);
    const data = client.getQueryData<InfiniteData<MessagesPage>>(keys.messages("c1"));
    expect(data?.pages[0].messages.map((m) => m.id)).toEqual(["m2", "m1"]);
  });

  it("updates read receipts", () => {
    const client = new QueryClient();
    seed(client);
    applyRealtimeEvent(client, { type: "conversation.read", data: { conversationId: "c1", readAt: "2026-03-15T11:00:00Z" } });
    expect(client.getQueryData<InfiniteData<MessagesPage>>(keys.messages("c1"))?.pages[0].otherLastReadAt).toBe("2026-03-15T11:00:00Z");
  });

  it("invalidates lists when a match appears or disappears", () => {
    const client = new QueryClient();
    const spy = jest.spyOn(client, "invalidateQueries");
    applyRealtimeEvent(client, { type: "match.new", data: { matchId: "m", conversationId: "c", userId: "u" } });
    expect(spy).toHaveBeenCalledWith({ queryKey: keys.matches });
    applyRealtimeEvent(client, { type: "match.removed", data: { matchId: "m" } });
    expect(spy).toHaveBeenCalledWith({ queryKey: keys.conversations });
  });
});

describe("reconnectDelay", () => {
  it("backs off exponentially up to a ceiling", () => {
    expect([0, 1, 2, 5, 20].map(reconnectDelay)).toEqual([1000, 2000, 4000, 30000, 30000]);
  });
});
