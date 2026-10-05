import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import type { Message } from "../api/types";
import { applyMatchRemoved, applyMessageNew, applyMessagesRead, appendMessage, queryKeys, type MessagesData } from "./cache";

const message = (id: string, sender: string, readAt?: string): Message => ({
  id,
  conversationId: "c1",
  senderUserId: sender,
  content: id,
  createdAt: "2026-01-01T10:00:00Z",
  readAt
});
const data = (...pages: Message[][]): MessagesData => ({
  pages: pages.map((messages) => ({ messages })),
  pageParams: pages.map(() => undefined)
});

describe("appendMessage", () => {
  it("adds the message to the newest page only", () => {
    const next = appendMessage(data([message("2", "a")], [message("1", "a")]), message("3", "b"))!;
    expect(next.pages[0].messages.map((m) => m.id)).toEqual(["2", "3"]);
    expect(next.pages[1].messages.map((m) => m.id)).toEqual(["1"]);
  });

  it("ignores a message that is already cached (realtime echo of our own send)", () => {
    const current = data([message("2", "a")]);
    expect(appendMessage(current, message("2", "a"))).toBe(current);
  });

  it("leaves an unloaded conversation alone", () => {
    expect(appendMessage(undefined, message("1", "a"))).toBeUndefined();
  });
});

describe("query cache updates", () => {
  it("applies a realtime message to the open conversation", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.messages("c1"), data([message("1", "a")]));
    applyMessageNew(client, message("2", "b"));
    expect(client.getQueryData<MessagesData>(queryKeys.messages("c1"))!.pages[0].messages).toHaveLength(2);
  });

  it("marks messages not sent by the reader as read", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.messages("c1"), data([message("1", "me"), message("2", "them")]));
    applyMessagesRead(client, "c1", "them", "2026-01-01T11:00:00Z");
    const [mine, theirs] = client.getQueryData<MessagesData>(queryKeys.messages("c1"))!.pages[0].messages;
    expect(mine.readAt).toBe("2026-01-01T11:00:00Z");
    expect(theirs.readAt).toBeUndefined();
  });

  it("drops cached conversations when a match disappears", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.messages("c1"), data([message("1", "a")]));
    applyMatchRemoved(client);
    expect(client.getQueryData(queryKeys.messages("c1"))).toBeUndefined();
  });
});
