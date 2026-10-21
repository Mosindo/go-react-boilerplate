import { describe, expect, it } from "vitest";
import { ageOn, clampAgeRange, parseBirthDate, validateEmail, validatePassword } from "./validation";

const now = new Date(2026, 5, 15);

describe("validateEmail", () => {
  it("accepts and rejects", () => {
    expect(validateEmail("  Jane@Example.com ")).toBeNull();
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("jane@")).not.toBeNull();
    expect(validateEmail("jane example@x.com")).not.toBeNull();
  });
});

describe("validatePassword", () => {
  it("enforces 8 chars and bcrypt's 72 bytes", () => {
    expect(validatePassword("1234567")).not.toBeNull();
    expect(validatePassword("12345678")).toBeNull();
    expect(validatePassword("é".repeat(40))).not.toBeNull();
  });
});

describe("parseBirthDate", () => {
  it("accepts adults and returns an ISO date", () => {
    const r = parseBirthDate("15", "6", "1994", now);
    expect(r).toEqual({ ok: true, iso: "1994-06-15", age: 32 });
  });
  it("is exact on the 18th birthday", () => {
    expect(parseBirthDate("15", "6", "2008", now).ok).toBe(true);
    expect(parseBirthDate("16", "6", "2008", now).ok).toBe(false);
  });
  it("rejects impossible or incomplete dates", () => {
    expect(parseBirthDate("31", "2", "1990", now).ok).toBe(false);
    expect(parseBirthDate("", "2", "1990", now).ok).toBe(false);
    expect(parseBirthDate("1", "1", "90", now).ok).toBe(false);
    expect(parseBirthDate("1", "1", "1900", now).ok).toBe(false);
  });
  it("handles leap days", () => {
    expect(parseBirthDate("29", "2", "2000", now).ok).toBe(true);
    expect(parseBirthDate("29", "2", "2001", now).ok).toBe(false);
  });
});

describe("ageOn", () => {
  it("counts full years", () => {
    expect(ageOn(2000, 6, 16, now)).toBe(25);
    expect(ageOn(2000, 6, 15, now)).toBe(26);
  });
});

describe("clampAgeRange", () => {
  it("keeps min <= max within bounds", () => {
    expect(clampAgeRange(40, 30, "min")).toEqual([40, 40]);
    expect(clampAgeRange(40, 30, "max")).toEqual([30, 30]);
    expect(clampAgeRange(10, 120, "min")).toEqual([18, 99]);
  });
});
