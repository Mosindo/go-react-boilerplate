import { describe, expect, it } from "vitest";
import { ageFromIso, parseBirthDate } from "./birthDate";
import { formatDistance, formatListTime, formatRelative } from "./format";
import { isRead, mergeMessage } from "./messages";
import { decideSwipe } from "./swipe";
import { normalizeEmail, validateEmail, validatePassword } from "./validation";
import type { Message } from "../api/types";

const NOW = new Date("2026-06-15T12:00:00Z");

describe("birth date", () => {
  it("computes age on the birthday boundary", () => {
    expect(ageFromIso("2008-06-15", NOW)).toBe(18);
    expect(ageFromIso("2008-06-16", NOW)).toBe(17);
  });
  it("accepts adults and returns an ISO date", () => {
    expect(parseBirthDate("5", "3", "1994", NOW)).toEqual({ iso: "1994-03-05", error: null });
  });
  it("rejects minors, impossible dates and malformed input", () => {
    expect(parseBirthDate("16", "6", "2008", NOW).error).toMatch(/18 ans/);
    expect(parseBirthDate("31", "2", "1990", NOW).error).toMatch(/n'existe pas/);
    expect(parseBirthDate("aa", "2", "1990", NOW).error).toMatch(/JJ/);
    expect(parseBirthDate("1", "1", "1900", NOW).error).toBeTruthy();
  });
});

describe("format", () => {
  it("formats distances approximately", () => {
    expect(formatDistance(undefined)).toBeNull();
    expect(formatDistance(1)).toBe("À moins de 1 km");
    expect(formatDistance(12)).toBe("À 12 km");
  });
  it("formats list times relative to now", () => {
    expect(formatListTime("2026-06-14T10:00:00Z", NOW)).toBe("Hier");
    expect(formatRelative("2026-06-15T11:59:50Z", NOW)).toBe("à l'instant");
    expect(formatRelative("2026-06-15T11:30:00Z", NOW)).toBe("il y a 30 min");
    expect(formatRelative("2026-06-15T09:00:00Z", NOW)).toBe("il y a 3 h");
  });
});

describe("validation", () => {
  it("normalizes and validates e-mail", () => {
    expect(normalizeEmail("  Foo@Bar.COM ")).toBe("foo@bar.com");
    expect(validateEmail("foo@bar.com")).toBeNull();
    expect(validateEmail("foo@bar")).not.toBeNull();
  });
  it("enforces password length in bytes", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("é".repeat(40))).not.toBeNull();
  });
});

describe("messages", () => {
  const m = (id: string, at: string, sender = "a"): Message => ({
    id,
    conversationId: "c",
    senderId: sender,
    body: id,
    createdAt: at
  });
  it("merges without duplicates, newest first", () => {
    const list = [m("2", "2026-01-02T00:00:00Z"), m("1", "2026-01-01T00:00:00Z")];
    expect(mergeMessage(list, m("2", "2026-01-02T00:00:00Z"))).toBe(list);
    expect(mergeMessage(list, m("3", "2026-01-03T00:00:00Z")).map((x) => x.id)).toEqual(["3", "2", "1"]);
  });
  it("computes read state for my messages only", () => {
    const msg = m("1", "2026-01-01T10:00:00Z", "me");
    expect(isRead(msg, "me", "2026-01-01T10:00:01Z")).toBe(true);
    expect(isRead(msg, "me", "2026-01-01T09:59:00Z")).toBe(false);
    expect(isRead(m("2", "2026-01-01T10:00:00Z", "peer"), "me", "2030-01-01T00:00:00Z")).toBe(false);
  });
});

describe("swipe gesture", () => {
  it("swipes on distance or on a fast flick", () => {
    expect(decideSwipe(150, 10, 0)).toBe("like");
    expect(decideSwipe(-150, 10, 0)).toBe("pass");
    expect(decideSwipe(60, 0, 1.2)).toBe("like");
    expect(decideSwipe(-60, 0, -1.2)).toBe("pass");
  });
  it("ignores small or vertical movements", () => {
    expect(decideSwipe(30, 0, 0.1)).toBeNull();
    expect(decideSwipe(200, 400, 0)).toBeNull();
  });
});
