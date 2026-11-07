import { describe, expect, it } from "vitest";
import { formatDistance, formatTime, isReadByPeer } from "./format";
import { ageFromBirthDate, validateBirthDate, validateEmail, validatePassword } from "./validation";

const now = new Date("2026-06-15T12:00:00Z");

describe("validation", () => {
  it("computes age on the birthday boundary", () => {
    expect(ageFromBirthDate("2008-06-15", now)).toBe(18);
    expect(ageFromBirthDate("2008-06-16", now)).toBe(17);
    expect(ageFromBirthDate("2008-02-30", now)).toBeNull();
    expect(ageFromBirthDate("nope", now)).toBeNull();
  });
  it("rejects minors and nonsense dates", () => {
    expect(validateBirthDate("2008-06-16", now)).toMatch(/18 ans/);
    expect(validateBirthDate("1900-01-01", now)).toBeTruthy();
    expect(validateBirthDate("1990-01-01", now)).toBeNull();
  });
  it("validates email and password", () => {
    expect(validateEmail("a@b.co")).toBeNull();
    expect(validateEmail("a@b")).toBeTruthy();
    expect(validatePassword("short")).toBeTruthy();
    expect(validatePassword("long-enough-1")).toBeNull();
    expect(validatePassword("é".repeat(40))).toBeTruthy();
  });
});

describe("format", () => {
  it("formats approximate distance", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("à moins de 1 km");
    expect(formatDistance(15)).toBe("à 15 km");
  });
  it("formats times relative to now", () => {
    expect(formatTime("2026-06-15T08:05:00", new Date("2026-06-15T20:00:00"))).toMatch(/08:05/);
    expect(formatTime("garbage", now)).toBe("");
  });
  it("derives read receipts", () => {
    expect(isReadByPeer("2026-01-01T10:00:00Z", null)).toBe(false);
    expect(isReadByPeer("2026-01-01T10:00:00Z", "2026-01-01T09:00:00Z")).toBe(false);
    expect(isReadByPeer("2026-01-01T10:00:00Z", "2026-01-01T10:00:00Z")).toBe(true);
  });
});
