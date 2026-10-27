import type { InfiniteData } from "@tanstack/react-query";
import type { MessagePage } from "../api/types";
import { mergeMessage } from "../state/realtime";

const msg = (id: number) => ({
  id,
  conversationId: "c",
  senderId: "u",
  body: `m${id}`,
  createdAt: "2026-01-01T00:00:00Z"
});

describe("mergeMessage", () => {
  const data: InfiniteData<MessagePage> = {
    pages: [{ messages: [msg(2), msg(1)], otherLastReadId: 0 }],
    pageParams: [undefined]
  };

  it("prepends a new message to the newest page", () => {
    const next = mergeMessage(data, msg(3));
    expect(next?.pages[0].messages.map((m) => m.id)).toEqual([3, 2, 1]);
  });

  it("ignores duplicates (REST response + websocket echo)", () => {
    expect(mergeMessage(data, msg(2))).toBe(data);
  });

  it("leaves an uncached conversation alone", () => {
    expect(mergeMessage(undefined, msg(1))).toBeUndefined();
  });
});
