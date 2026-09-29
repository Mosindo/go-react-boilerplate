import { describe, expect, it } from "vitest";
import type { Message } from "../../api/models";
import {
  buildThreadItems,
  canSend,
  classifySendFailure,
  flattenMessages,
  markMineRead,
  showCounter,
  upsertMessage,
  type MessagesPage,
  type PendingMessage
} from "./messages";

function msg(id: string, senderId: string, at: Date, readAt: string | null = null): Message {
  return { id, conversationId: "c1", senderId, body: `body ${id}`, createdAt: at.toISOString(), readAt };
}

const me = "me";
const them = "them";
const now = new Date(2026, 5, 15, 12);

describe("flattenMessages / upsertMessage", () => {
  const m1 = msg("m1", them, new Date(2026, 5, 15, 9));
  const m2 = msg("m2", me, new Date(2026, 5, 15, 10));
  const data = { pages: [{ messages: [m2], nextCursor: "m2" }, { messages: [m1], nextCursor: null }] };

  it("dedupes across pages and sorts newest first", () => {
    const pages: MessagesPage[] = [
      { messages: [m1, m2], nextCursor: null },
      { messages: [m1], nextCursor: null }
    ];
    expect(flattenMessages(pages).map((m) => m.id)).toEqual(["m2", "m1"]);
  });

  it("adds a new message at the front and is idempotent", () => {
    const m3 = msg("m3", them, new Date(2026, 5, 15, 11));
    const once = upsertMessage(data, m3);
    expect(flattenMessages(once.pages).map((m) => m.id)).toEqual(["m3", "m2", "m1"]);
    const twice = upsertMessage(once, m3);
    expect(flattenMessages(twice.pages)).toHaveLength(3);
  });

  it("updates an existing message in place", () => {
    const read = { ...m2, readAt: "2026-06-15T10:05:00.000Z" };
    const next = upsertMessage(data, read);
    expect(flattenMessages(next.pages)[0].readAt).toBe(read.readAt);
  });

  it("creates a page when the cache is empty", () => {
    const next = upsertMessage({ pages: [] as MessagesPage[] }, m1);
    expect(next.pages[0].messages).toEqual([m1]);
  });
});

describe("markMineRead", () => {
  it("only touches my unread messages", () => {
    const data = {
      pages: [
        {
          messages: [msg("a", me, new Date(2026, 5, 15, 10)), msg("b", them, new Date(2026, 5, 15, 9))],
          nextCursor: null
        }
      ]
    };
    const next = markMineRead(data, me, "2026-06-15T11:00:00.000Z");
    expect(next.pages[0].messages[0].readAt).toBe("2026-06-15T11:00:00.000Z");
    expect(next.pages[0].messages[1].readAt).toBeNull();
    expect(markMineRead(next, me, "2026-06-15T12:00:00.000Z")).toBe(next);
  });
});

describe("buildThreadItems", () => {
  it("puts pending first and inserts separators above the oldest message of each day", () => {
    const server = [
      msg("m3", them, new Date(2026, 5, 15, 10)),
      msg("m2", them, new Date(2026, 5, 15, 9)),
      msg("m1", me, new Date(2026, 5, 14, 22))
    ];
    const pending: PendingMessage[] = [
      { localId: "l1", body: "hey", createdAt: new Date(2026, 5, 15, 11).toISOString(), status: "sending" }
    ];
    const items = buildThreadItems(server, pending, me, now);
    expect(items.map((i) => (i.kind === "message" ? i.key : `sep:${i.label}`))).toEqual([
      "l1",
      "m3",
      "m2",
      "sep:Today",
      "m1",
      "sep:Yesterday"
    ]);
    const m3 = items[1];
    expect(m3.kind === "message" && m3.groupedWithOlder).toBe(true);
    const l1 = items[0];
    expect(l1.kind === "message" && l1.pending).toBe("sending");
    expect(l1.kind === "message" && l1.mine).toBe(true);
  });

  it("returns an empty list for an empty thread", () => {
    expect(buildThreadItems([], [], me, now)).toEqual([]);
  });
});

describe("composer rules", () => {
  it("enables send only for non-blank text within the limit", () => {
    expect(canSend("   ")).toBe(false);
    expect(canSend("hi")).toBe(true);
    expect(canSend("a".repeat(2001))).toBe(false);
    expect(showCounter(1799)).toBe(false);
    expect(showCounter(1800)).toBe(true);
  });
  it("classifies failures", () => {
    expect(classifySendFailure(403)).toBe("forbidden");
    expect(classifySendFailure(404)).toBe("gone");
    expect(classifySendFailure(null)).toBe("retry");
  });
});
