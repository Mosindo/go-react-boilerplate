import { describe, expect, it } from "vitest";
import {
  ageFromIso,
  clampAgeRange,
  isoToParts,
  toIsoDate,
  utf8Length,
  validateBirthDate,
  validateEmail,
  validatePassword
} from "../validation";

const today = new Date(Date.UTC(2026, 5, 15));

describe("validateEmail", () => {
  it("accepts and normalises plausible addresses", () => {
    expect(validateEmail("  Ada@Example.COM ")).toBeNull();
  });
  it("rejects malformed addresses", () => {
    for (const value of ["", "ada", "ada@", "@example.com", "ada@example", "a b@example.com"]) {
      expect(validateEmail(value)).not.toBeNull();
    }
  });
});

describe("validatePassword", () => {
  it("enforces the bcrypt-aware byte limits", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long enough")).toBeNull();
    expect(validatePassword("é".repeat(37))).not.toBeNull(); // 74 bytes
    expect(utf8Length("é")).toBe(2);
    expect(utf8Length("😀")).toBe(4);
  });
});

describe("birth date", () => {
  it("builds ISO dates only for real calendar dates", () => {
    expect(toIsoDate({ day: "5", month: "3", year: "1990" })).toBe("1990-03-05");
    expect(toIsoDate({ day: "31", month: "2", year: "1990" })).toBeNull();
    expect(toIsoDate({ day: "29", month: "2", year: "2001" })).toBeNull();
    expect(toIsoDate({ day: "29", month: "2", year: "2000" })).toBe("2000-02-29");
    expect(toIsoDate({ day: "aa", month: "1", year: "1990" })).toBeNull();
    expect(toIsoDate({ day: "1", month: "1", year: "90" })).toBeNull();
  });

  it("computes age on the birthday boundary", () => {
    expect(ageFromIso("2008-06-15", today)).toBe(18);
    expect(ageFromIso("2008-06-16", today)).toBe(17);
  });

  it("blocks minors and absurd dates", () => {
    expect(validateBirthDate({ day: "16", month: "6", year: "2008" }, today)).toMatch(/18 ans/);
    expect(validateBirthDate({ day: "15", month: "6", year: "2008" }, today)).toBeNull();
    expect(validateBirthDate({ day: "1", month: "1", year: "1890" }, today)).not.toBeNull();
    expect(validateBirthDate({ day: "x", month: "1", year: "1990" }, today)).not.toBeNull();
  });

  it("round-trips through ISO parts", () => {
    expect(isoToParts("1990-03-05")).toEqual({ day: "05", month: "03", year: "1990" });
  });
});

describe("clampAgeRange", () => {
  it("keeps the range inside 18-99 and ordered", () => {
    expect(clampAgeRange(10, 120)).toEqual({ min: 18, max: 99 });
    expect(clampAgeRange(40, 30)).toEqual({ min: 30, max: 40 });
  });
});
