import {
  buildChatItems,
  dayLabel,
  flattenPages,
  formatClock,
  formatListTime,
  lastSeenMessageKey,
  mergeWithPending,
  upsertMessage,
  validateDraft,
  type PendingMessage
} from "./messages";
import type { Message } from "../types";

function msg(id: string, senderId: string, createdAt: Date, content = id): Message {
  return { id, conversationId: "c1", senderId, content, createdAt: createdAt.toISOString() };
}

const now = new Date(2026, 9, 7, 15, 0, 0);

describe("flattenPages / upsertMessage", () => {
  it("dedupes across pages and sorts newest first", () => {
    const a = msg("a", "me", new Date(2026, 9, 7, 10));
    const b = msg("b", "peer", new Date(2026, 9, 7, 11));
    const c = msg("c", "me", new Date(2026, 9, 7, 12));
    const out = flattenPages([{ items: [c, b] }, { items: [b, a] }]);
    expect(out.map((m) => m.id)).toEqual(["c", "b", "a"]);
  });
  it("upsert inserts in order, ignores identical, replaces changed", () => {
    const a = msg("a", "me", new Date(2026, 9, 7, 10));
    const c = msg("c", "me", new Date(2026, 9, 7, 12));
    const list = [c, a];
    const b = msg("b", "peer", new Date(2026, 9, 7, 11));
    expect(upsertMessage(list, b).map((m) => m.id)).toEqual(["c", "b", "a"]);
    expect(upsertMessage(list, a)).toBe(list);
    expect(upsertMessage(list, { ...a, content: "edited" })[1].content).toBe("edited");
  });
});

describe("mergeWithPending", () => {
  const base: PendingMessage = {
    localId: "local-1",
    conversationId: "c1",
    senderId: "me",
    content: "hello",
    createdAt: new Date(2026, 9, 7, 12, 0, 0).toISOString(),
    status: "sending"
  };
  it("keeps unsent messages on top", () => {
    const server = [msg("a", "peer", new Date(2026, 9, 7, 11))];
    const out = mergeWithPending(server, [base]);
    expect(out.map((v) => v.key)).toEqual(["local-1", "a"]);
    expect(out[0].status).toBe("sending");
  });
  it("drops a sending message once the server echo arrives", () => {
    const echo = msg("srv-1", "me", new Date(2026, 9, 7, 12, 0, 1), "hello");
    const out = mergeWithPending([echo], [base]);
    expect(out.map((v) => v.key)).toEqual(["srv-1"]);
  });
  it("claims each server message once", () => {
    const echo = msg("srv-1", "me", new Date(2026, 9, 7, 12, 0, 1), "hello");
    const second = { ...base, localId: "local-2" };
    const out = mergeWithPending([echo], [base, second]);
    expect(out.map((v) => v.key).sort()).toEqual(["local-2", "srv-1"]);
  });
  it("never reconciles failed messages automatically", () => {
    const echo = msg("srv-1", "me", new Date(2026, 9, 7, 12, 0, 1), "hello");
    const out = mergeWithPending([echo], [{ ...base, status: "failed" }]);
    expect(out).toHaveLength(2);
    expect(out[0].status).toBe("failed");
  });
  it("does not match other senders or other content", () => {
    const other = msg("srv-1", "peer", new Date(2026, 9, 7, 12, 0, 1), "hello");
    expect(mergeWithPending([other], [base])).toHaveLength(2);
  });
});

describe("lastSeenMessageKey", () => {
  const views = mergeWithPending(
    [
      msg("m3", "peer", new Date(2026, 9, 7, 12)),
      msg("m2", "me", new Date(2026, 9, 7, 11)),
      msg("m1", "me", new Date(2026, 9, 7, 10))
    ],
    []
  );
  it("returns my newest message read by the peer", () => {
    expect(lastSeenMessageKey(views, "me", new Date(2026, 9, 7, 11, 30).toISOString())).toBe("m2");
    expect(lastSeenMessageKey(views, "me", new Date(2026, 9, 7, 10, 30).toISOString())).toBe("m1");
  });
  it("returns null when unread or unknown", () => {
    expect(lastSeenMessageKey(views, "me", null)).toBeNull();
    expect(lastSeenMessageKey(views, "me", new Date(2026, 9, 7, 9).toISOString())).toBeNull();
  });
});

describe("date labels and grouping", () => {
  it("labels today, yesterday, this year and older", () => {
    expect(dayLabel(new Date(2026, 9, 7, 1).toISOString(), now)).toBe("Today");
    expect(dayLabel(new Date(2026, 9, 6, 23).toISOString(), now)).toBe("Yesterday");
    expect(dayLabel(new Date(2026, 9, 1, 9).toISOString(), now)).toBe("Thu, Oct 1");
    expect(dayLabel(new Date(2025, 11, 25, 9).toISOString(), now)).toBe("Thu, Dec 25, 2025");
  });
  it("formats clock and list times", () => {
    expect(formatClock(new Date(2026, 9, 7, 0, 5).toISOString())).toBe("12:05 AM");
    expect(formatClock(new Date(2026, 9, 7, 15, 30).toISOString())).toBe("3:30 PM");
    expect(formatListTime(new Date(2026, 9, 7, 9, 0).toISOString(), now)).toBe("9:00 AM");
    expect(formatListTime(new Date(2026, 9, 6, 9, 0).toISOString(), now)).toBe("Yesterday");
    expect(formatListTime(new Date(2026, 9, 3, 9, 0).toISOString(), now)).toBe("Sat");
    expect(formatListTime(new Date(2026, 8, 1, 9, 0).toISOString(), now)).toBe("Sep 1");
  });
  it("places a separator above the oldest message of each day", () => {
    const views = mergeWithPending(
      [
        msg("d", "peer", new Date(2026, 9, 7, 9, 0)),
        msg("c", "me", new Date(2026, 9, 6, 22, 0)),
        msg("b", "me", new Date(2026, 9, 6, 21, 59, 30)),
        msg("a", "peer", new Date(2026, 9, 5, 8, 0))
      ],
      []
    );
    const items = buildChatItems(views, "me", null, now);
    expect(items.map((i) => (i.kind === "separator" ? `sep:${i.label}` : i.key))).toEqual([
      "d",
      "sep:Today",
      "c",
      "b",
      "sep:Yesterday",
      "a",
      "sep:Mon, Oct 5"
    ]);
  });
  it("marks seen on one message and hides time inside a same-minute run", () => {
    const views = mergeWithPending(
      [msg("b", "me", new Date(2026, 9, 7, 12, 0, 30)), msg("a", "me", new Date(2026, 9, 7, 12, 0, 0))],
      []
    );
    const items = buildChatItems(views, "me", new Date(2026, 9, 7, 13).toISOString(), now);
    const messages = items.filter((i) => i.kind === "message");
    expect(messages.map((i) => i.kind === "message" && i.showSeen)).toEqual([true, false]);
    expect(messages.map((i) => i.kind === "message" && i.showTime)).toEqual([true, false]);
  });
});

describe("validateDraft", () => {
  it("trims and enforces limits", () => {
    expect(validateDraft("  hi  ")).toEqual({ ok: true, content: "hi" });
    expect(validateDraft("   ").ok).toBe(false);
    expect(validateDraft("x".repeat(2000)).ok).toBe(true);
    expect(validateDraft("x".repeat(2001)).ok).toBe(false);
  });
});
