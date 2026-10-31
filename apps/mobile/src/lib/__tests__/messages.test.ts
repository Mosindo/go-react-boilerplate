import type { Message, MessagesPage } from "../../api/types";
import { flattenMessages, prependMessage } from "../messages";

const msg = (id: string): Message => ({ id, conversationId: "c", senderId: "u", body: id, createdAt: "2026-01-01T00:00:00Z" });

describe("prependMessage", () => {
  it("creates the first page when the cache is empty", () => {
    expect(flattenMessages(prependMessage([], msg("a"))).map((m) => m.id)).toEqual(["a"]);
  });

  it("puts new messages first", () => {
    const pages: MessagesPage[] = [{ messages: [msg("a")] }, { messages: [msg("z")] }];
    expect(flattenMessages(prependMessage(pages, msg("b"))).map((m) => m.id)).toEqual(["b", "a", "z"]);
  });

  it("ignores duplicates (REST response + WebSocket echo)", () => {
    const pages: MessagesPage[] = [{ messages: [msg("a")] }];
    expect(prependMessage(pages, msg("a"))).toBe(pages);
  });
});
