import type { ChatMessage } from "../../api/types";
import { buildChatRows, formatClock, formatDayLabel, formatListTime, receiptGlyph, receiptLabel } from "../chat";

const now = new Date(2026, 5, 15, 12, 0, 0);

function message(id: string, senderId: string, at: Date, extra: Partial<ChatMessage> = {}): ChatMessage {
  return { id, conversationId: "c1", senderId, body: id, createdAt: at.toISOString(), readAt: null, ...extra };
}

describe("time formatting", () => {
  it("formats the clock with zero padding", () => {
    expect(formatClock(new Date(2026, 0, 1, 9, 5))).toBe("09:05");
  });

  it("labels days relative to now", () => {
    expect(formatDayLabel(new Date(2026, 5, 15, 8).toISOString(), now)).toBe("Today");
    expect(formatDayLabel(new Date(2026, 5, 14, 23).toISOString(), now)).toBe("Yesterday");
    expect(formatDayLabel(new Date(2026, 5, 11, 10).toISOString(), now)).toBe("Thursday");
    expect(formatDayLabel(new Date(2026, 4, 1, 10).toISOString(), now)).toBe("1 May");
    expect(formatDayLabel(new Date(2025, 4, 1, 10).toISOString(), now)).toBe("1 May 2025");
  });

  it("uses the clock for today and the day label otherwise in lists", () => {
    expect(formatListTime(new Date(2026, 5, 15, 7, 30).toISOString(), now)).toBe("07:30");
    expect(formatListTime(new Date(2026, 5, 14, 7, 30).toISOString(), now)).toBe("Yesterday");
  });
});

describe("buildChatRows", () => {
  it("groups bursts by sender and shows the time on the newest message of a burst", () => {
    const base = new Date(2026, 5, 15, 10, 0, 0);
    const newestFirst = [
      message("m3", "me", new Date(base.getTime() + 60_000)),
      message("m2", "me", new Date(base.getTime() + 30_000)),
      message("m1", "them", base)
    ];
    const rows = buildChatRows(newestFirst, now);
    const messages = rows.filter((row) => row.kind === "message");
    expect(messages.map((row) => row.kind === "message" && [row.message.id, row.showTime, row.startsGroup])).toEqual([
      ["m3", true, false],
      ["m2", false, true],
      ["m1", true, true]
    ]);
  });

  it("starts a new group after a long pause", () => {
    const base = new Date(2026, 5, 15, 10, 0, 0);
    const rows = buildChatRows(
      [message("b", "me", new Date(base.getTime() + 10 * 60_000)), message("a", "me", base)],
      now
    );
    const flags = rows.flatMap((row) => (row.kind === "message" ? [[row.showTime, row.startsGroup]] : []));
    expect(flags).toEqual([
      [true, true],
      [true, true]
    ]);
  });

  it("inserts a day separator above the first message of each day", () => {
    const rows = buildChatRows(
      [message("today", "me", new Date(2026, 5, 15, 9)), message("yesterday", "them", new Date(2026, 5, 14, 20))],
      now
    );
    expect(rows.map((row) => (row.kind === "day" ? row.label : row.key))).toEqual([
      "today",
      "Today",
      "yesterday",
      "Yesterday"
    ]);
  });

  it("returns nothing for an empty thread", () => {
    expect(buildChatRows([], now)).toEqual([]);
  });
});

describe("receipts", () => {
  const base = message("x", "me", now);
  it("maps message state to glyphs and labels", () => {
    expect(receiptGlyph({ ...base, status: "sending" })).toBe("…");
    expect(receiptGlyph({ ...base, status: "failed" })).toBe("!");
    expect(receiptGlyph(base)).toBe("✓");
    expect(receiptGlyph({ ...base, readAt: now.toISOString() })).toBe("✓✓");
    expect(receiptLabel(base)).toBe("Delivered");
    expect(receiptLabel({ ...base, readAt: now.toISOString() })).toBe("Read");
    expect(receiptLabel({ ...base, status: "failed" })).toMatch(/retry/);
  });
});
