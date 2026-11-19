import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import type { Message, MessagesPage } from "../api/types";
import { addMessageToCache, markReadInCache } from "./realtime";

jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn(),
  setItemAsync: jest.fn(),
  deleteItemAsync: jest.fn()
}));

const msg = (id: string, createdAt: string, isMine = true): Message => ({
  id,
  conversationId: "c1",
  senderId: isMine ? "me" : "them",
  body: id,
  createdAt,
  isMine,
  isRead: false
});

function seed(client: QueryClient, messages: Message[]) {
  client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages("c1"), {
    pages: [{ messages, nextCursor: null, partnerLastReadAt: null }],
    pageParams: [null]
  });
}

const read = (client: QueryClient) => client.getQueryData<InfiniteData<MessagesPage>>(queryKeys.messages("c1"))!.pages[0].messages;

// gcTime: Infinity avoids scheduling garbage-collection timers that would keep Jest alive.
const clients: QueryClient[] = [];
const newClient = () => {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity } } });
  clients.push(client);
  return client;
};
afterEach(() => clients.splice(0).forEach((c) => c.clear()));

describe("realtime cache helpers", () => {
  it("prepends new messages once", () => {
    const client = newClient();
    seed(client, [msg("a", "2026-01-01T10:00:00Z")]);
    const b = msg("b", "2026-01-01T10:01:00Z", false);
    addMessageToCache(client, b);
    addMessageToCache(client, b);
    expect(read(client).map((m) => m.id)).toEqual(["b", "a"]);
  });

  it("ignores conversations that are not loaded", () => {
    const client = newClient();
    addMessageToCache(client, msg("x", "2026-01-01T10:00:00Z"));
    expect(client.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });

  it("marks my messages read up to the read marker", () => {
    const client = newClient();
    seed(client, [msg("new", "2026-01-01T10:05:00Z"), msg("old", "2026-01-01T10:00:00Z"), msg("theirs", "2026-01-01T09:00:00Z", false)]);
    markReadInCache(client, "c1", "2026-01-01T10:01:00Z");
    const [newer, older, theirs] = read(client);
    expect(newer.isRead).toBe(false);
    expect(older.isRead).toBe(true);
    expect(theirs.isRead).toBe(false);
  });
});
