import { describe, expect, it } from "vitest";
import { backoffDelay } from "./backoff";
import { ageFromIso, formatBirthInput, maskBirthInput, parseBirthInput, validateBirthInput } from "./date";
import { formatDistance, formatRelative } from "./format";
import { isRead, mergePages, upsertMessage } from "./messages";
import { validateEmail, validatePassword } from "./validation";
import type { Message } from "../api/types";

const msg = (id: string, at: string): Message => ({
  id,
  conversationId: "c",
  senderId: "u",
  body: id,
  createdAt: at,
});

describe("date", () => {
  it("parses and validates real calendar dates", () => {
    expect(parseBirthInput("29/02/2000")).toBe("2000-02-29");
    expect(parseBirthInput("29/02/2001")).toBeNull();
    expect(parseBirthInput("31/04/1990")).toBeNull();
    expect(parseBirthInput("1990-01-01")).toBeNull();
  });
  it("formats and masks input", () => {
    expect(formatBirthInput("1995-04-12")).toBe("12/04/1995");
    expect(maskBirthInput("12041995")).toBe("12/04/1995");
    expect(maskBirthInput("1a2b0")).toBe("12/0");
  });
  it("enforces the 18+ rule on the exact birthday", () => {
    const now = new Date(2026, 5, 15);
    expect(ageFromIso("2008-06-15", now)).toBe(18);
    expect(ageFromIso("2008-06-16", now)).toBe(17);
    expect(validateBirthInput("15/06/2008", now)).toBeNull();
    expect(validateBirthInput("16/06/2008", now)).toMatch(/18 ans/);
    expect(validateBirthInput("01/01/1900", now)).not.toBeNull();
  });
});

describe("validation", () => {
  it("validates emails", () => {
    expect(validateEmail(" A@b.co ")).toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
  });
  it("mirrors the server password policy", () => {
    expect(validatePassword("Passw0rd")).toBeNull();
    expect(validatePassword("short1")).not.toBeNull();
    expect(validatePassword("onlyletters")).not.toBeNull();
    expect(validatePassword("12345678")).not.toBeNull();
  });
});

describe("messages", () => {
  it("upserts without duplicates, newest first", () => {
    const a = msg("a", "2026-01-01T10:00:00Z");
    const b = msg("b", "2026-01-01T11:00:00Z");
    const list = upsertMessage(upsertMessage([], a), b);
    expect(list.map((m) => m.id)).toEqual(["b", "a"]);
    expect(upsertMessage(list, a)).toBe(list);
  });
  it("merges pages and dedupes", () => {
    const a = msg("a", "2026-01-01T10:00:00Z");
    const b = msg("b", "2026-01-01T09:00:00Z");
    expect(mergePages([[a], [a, b]]).map((m) => m.id)).toEqual(["a", "b"]);
  });
  it("derives read state", () => {
    const m = msg("a", "2026-01-01T10:00:00Z");
    expect(isRead(m, "2026-01-01T10:00:01Z")).toBe(true);
    expect(isRead(m, "2026-01-01T09:59:59Z")).toBe(false);
  });
});

describe("format", () => {
  it("formats relative times", () => {
    const now = new Date("2026-01-10T12:00:00Z");
    expect(formatRelative("2026-01-10T11:59:50Z", now)).toBe("À l'instant");
    expect(formatRelative("2026-01-10T11:30:00Z", now)).toBe("30 min");
    expect(formatRelative("2026-01-10T09:00:00Z", now)).toBe("3 h");
    expect(formatRelative("2026-01-09T09:00:00Z", now)).toBe("Hier");
  });
  it("describes distances approximately", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(5)).toBe("À moins de 5 km");
    expect(formatDistance(40)).toBe("À environ 40 km");
  });
});

describe("backoff", () => {
  it("grows exponentially and is capped", () => {
    expect(backoffDelay(0)).toBe(1000);
    expect(backoffDelay(3)).toBe(8000);
    expect(backoffDelay(20)).toBe(30_000);
  });
});
