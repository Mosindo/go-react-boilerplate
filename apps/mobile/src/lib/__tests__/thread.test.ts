import { describe, expect, it } from "vitest";
import type { Message } from "../../api/types";
import { buildThreadRows, hasUnreadIncoming, prependPage } from "../thread";

const msg = (id: string, senderId: string, createdAt: string, readAt: string | null = null): Message => ({
  id,
  matchId: "m",
  senderId,
  body: id,
  createdAt,
  readAt
});

describe("buildThreadRows", () => {
  const messages = [
    msg("1", "me", "2026-06-14T10:00:00", "2026-06-14T10:05:00"),
    msg("2", "them", "2026-06-14T10:10:00"),
    msg("3", "me", "2026-06-15T09:00:00", "2026-06-15T09:30:00")
  ];

  it("returns newest first with a separator at the start of each day", () => {
    const rows = buildThreadRows(messages, "me", () => "jour");
    expect(rows.map((r) => (r.kind === "day" ? "day" : r.message.id))).toEqual(["3", "day", "2", "1", "day"]);
  });

  it("only shows the read receipt under my latest message", () => {
    const rows = buildThreadRows(messages, "me", () => "jour");
    const receipts = rows.filter((r) => r.kind === "message" && r.showReceipt).map((r) => r.kind === "message" && r.message.id);
    expect(receipts).toEqual(["3"]);
  });

  it("shows no receipt while my latest message is unread", () => {
    const rows = buildThreadRows([msg("1", "me", "2026-06-14T10:00:00")], "me", () => "jour");
    expect(rows.some((r) => r.kind === "message" && r.showReceipt)).toBe(false);
  });
});

describe("helpers", () => {
  it("detects unread incoming messages", () => {
    expect(hasUnreadIncoming([msg("1", "them", "2026-06-14T10:00:00")], "me")).toBe(true);
    expect(hasUnreadIncoming([msg("1", "them", "2026-06-14T10:00:00", "2026-06-14T10:01:00")], "me")).toBe(false);
    expect(hasUnreadIncoming([msg("1", "me", "2026-06-14T10:00:00")], "me")).toBe(false);
  });

  it("prepends older pages without duplicates", () => {
    const merged = prependPage([msg("3", "me", "c")], [msg("1", "me", "a"), msg("3", "me", "c")]);
    expect(merged.map((m) => m.id)).toEqual(["1", "3"]);
  });
});
