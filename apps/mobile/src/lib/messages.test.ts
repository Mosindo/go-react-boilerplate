import { describe, expect, it } from "vitest";
import type { ChatMessage } from "../api/types";
import { appendOlder, applyReadReceipt, mergeMessage } from "./messages";

const msg = (id: string, createdAt: string, senderId = "a", readAt: string | null = null): ChatMessage => ({ id, matchId: "m", senderId, body: id, createdAt, readAt });

describe("mergeMessage", () => {
  it("keeps newest first and ignores duplicates (REST reply + WebSocket push)", () => {
    const list = [msg("2", "2026-01-01T10:00:02Z"), msg("1", "2026-01-01T10:00:01Z")];
    const merged = mergeMessage(list, msg("3", "2026-01-01T10:00:03Z"));
    expect(merged.map((m) => m.id)).toEqual(["3", "2", "1"]);
    expect(mergeMessage(merged, msg("3", "2026-01-01T10:00:03Z"))).toBe(merged);
  });
});

describe("applyReadReceipt", () => {
  it("marks only the other side's unread messages", () => {
    const list = [msg("2", "2026-01-01T10:00:02Z", "a"), msg("1", "2026-01-01T10:00:01Z", "b")];
    const out = applyReadReceipt(list, "b", "2026-01-01T11:00:00Z");
    expect(out[0].readAt).toBe("2026-01-01T11:00:00Z");
    expect(out[1].readAt).toBeNull();
  });
});

describe("appendOlder", () => {
  it("drops overlap when paginating backwards", () => {
    const list = [msg("2", "2026-01-01T10:00:02Z")];
    expect(appendOlder(list, [msg("2", "2026-01-01T10:00:02Z"), msg("1", "2026-01-01T10:00:01Z")]).map((m) => m.id)).toEqual(["2", "1"]);
  });
});
