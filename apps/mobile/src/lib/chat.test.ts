import { describe, expect, it } from "vitest";
import type { Message } from "../api/types";
import { visiblePending, type PendingMessage } from "./chat";

const me = "me";
const msg = (id: string, content: string, at: string, sender = me): Message => ({
  id,
  conversationId: "c",
  senderUserId: sender,
  content,
  createdAt: at
});
const pending = (tempId: string, content: string, at: string, failed = false): PendingMessage => ({
  tempId,
  content,
  createdAt: at,
  failed
});

describe("visiblePending", () => {
  it("hides a pending message once the server confirmed it", () => {
    const confirmed = [msg("1", "salut", "2026-01-01T10:00:01Z")];
    expect(visiblePending(confirmed, [pending("p1", "salut", "2026-01-01T10:00:00Z")], me)).toEqual([]);
  });

  it("keeps pending messages that are not confirmed yet", () => {
    expect(visiblePending([], [pending("p1", "salut", "2026-01-01T10:00:00Z")], me)).toHaveLength(1);
  });

  it("never hides a failed message", () => {
    const confirmed = [msg("1", "salut", "2026-01-01T10:00:01Z")];
    expect(visiblePending(confirmed, [pending("p1", "salut", "2026-01-01T10:00:00Z", true)], me)).toHaveLength(1);
  });

  it("does not match the other person's identical text or an old message", () => {
    const messages = [msg("1", "ok", "2026-01-01T10:00:01Z", "them"), msg("2", "ok", "2025-12-01T10:00:00Z")];
    expect(visiblePending(messages, [pending("p1", "ok", "2026-01-01T10:00:00Z")], me)).toHaveLength(1);
  });

  it("matches each confirmed message once so two identical quick messages stay two", () => {
    const messages = [msg("1", "ok", "2026-01-01T10:00:01Z")];
    const list = [pending("p1", "ok", "2026-01-01T10:00:00Z"), pending("p2", "ok", "2026-01-01T10:00:00.5Z")];
    expect(visiblePending(messages, list, me).map((p) => p.tempId)).toEqual(["p2"]);
  });
});
