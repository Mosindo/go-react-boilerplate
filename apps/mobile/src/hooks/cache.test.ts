import { insertMessage } from "./cache";
import type { Message, MessagesPage } from "../api/types";

const msg = (id: string): Message => ({ id, conversationId: "c", senderId: "u", body: id, createdAt: new Date().toISOString() });

describe("insertMessage", () => {
  it("prepends new messages to the first page and ignores duplicates", () => {
    const data = { pages: [{ messages: [msg("b")], nextCursor: null, peerLastReadAt: null } as MessagesPage], pageParams: [null] };
    const next = insertMessage(data, msg("c"));
    expect(next?.pages[0].messages.map((m) => m.id)).toEqual(["c", "b"]);
    expect(insertMessage(next, msg("c"))).toBe(next);
  });

  it("leaves an empty cache untouched", () => {
    expect(insertMessage(undefined, msg("a"))).toBeUndefined();
  });
});
