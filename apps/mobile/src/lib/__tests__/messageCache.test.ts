import { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../api/queryKeys";
import type { ChatMessage, Message } from "../../api/types";
import { addLocalMessage, removeMessage, resolveLocalMessage, setLocalStatus } from "../messageCache";
import type { CursorData } from "../pagination";

const read = (client: QueryClient) =>
  client.getQueryData<CursorData<"messages", ChatMessage>>(queryKeys.messages("c1"))?.pages[0].messages ?? [];

const saved: Message = {
  id: "srv1",
  conversationId: "c1",
  senderId: "me",
  body: "yo",
  createdAt: "2026-01-01T00:00:00Z",
  readAt: null
};

describe("optimistic messages", () => {
  it("adds a sending message, then swaps it for the saved one", () => {
    const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    const local = addLocalMessage(client, "c1", "me", "yo");
    expect(read(client)[0]).toMatchObject({ body: "yo", status: "sending" });
    resolveLocalMessage(client, local.id, saved);
    expect(read(client)).toEqual([saved]);
  });

  it("does not duplicate when the realtime echo arrived first", () => {
    const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    const local = addLocalMessage(client, "c1", "me", "yo");
    client.setQueryData<CursorData<"messages", ChatMessage>>(queryKeys.messages("c1"), (d) =>
      d ? { ...d, pages: [{ ...d.pages[0], messages: [saved, ...d.pages[0].messages] }] } : d
    );
    resolveLocalMessage(client, local.id, saved);
    expect(read(client).map((m) => m.id)).toEqual(["srv1"]);
  });

  it("marks failures, supports retry state and discard", () => {
    const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
    const local = addLocalMessage(client, "c1", "me", "yo");
    setLocalStatus(client, "c1", local.id, "failed");
    expect(read(client)[0].status).toBe("failed");
    setLocalStatus(client, "c1", local.id, "sending");
    expect(read(client)[0].status).toBe("sending");
    removeMessage(client, "c1", local.id);
    expect(read(client)).toEqual([]);
  });
});
