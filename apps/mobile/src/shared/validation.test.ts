import {
  calculateAge,
  normalizeEmail,
  parseBirthDate,
  splitIsoDate,
  utf8ByteLength,
  validateEmail,
  validatePassword,
  validatePasswordConfirmation
} from "./validation";

const today = new Date(2026, 5, 15); // 15 June 2026

describe("parseBirthDate", () => {
  it("accepts an adult and returns an ISO date and age", () => {
    expect(parseBirthDate("14", "06", "2008", today)).toEqual({ ok: true, isoDate: "2008-06-14", age: 18 });
  });

  it("rejects someone turning 18 tomorrow", () => {
    const result = parseBirthDate("16", "06", "2008", today);
    expect(result.ok).toBe(false);
  });

  it("accepts the 18th birthday itself", () => {
    expect(parseBirthDate("15", "06", "2008", today)).toMatchObject({ ok: true, age: 18 });
  });

  it("rejects minors, futures and ages above 99", () => {
    expect(parseBirthDate("01", "01", "2015", today).ok).toBe(false);
    expect(parseBirthDate("01", "01", "2030", today).ok).toBe(false);
    expect(parseBirthDate("01", "01", "1900", today).ok).toBe(false);
  });

  it("rejects impossible and incomplete dates", () => {
    expect(parseBirthDate("31", "02", "1990", today)).toEqual({ ok: false, error: "Cette date n'existe pas." });
    expect(parseBirthDate("29", "02", "2001", today).ok).toBe(false);
    expect(parseBirthDate("29", "02", "2000", today).ok).toBe(true);
    expect(parseBirthDate("", "01", "1990", today).ok).toBe(false);
    expect(parseBirthDate("1", "1", "90", today).ok).toBe(false);
  });

  it("zero-pads the ISO output", () => {
    expect(parseBirthDate("3", "7", "1990", today)).toMatchObject({ isoDate: "1990-07-03" });
  });
});

describe("calculateAge / splitIsoDate", () => {
  it("counts birthdays precisely", () => {
    expect(calculateAge(1990, 6, 16, today)).toBe(35);
    expect(calculateAge(1990, 6, 15, today)).toBe(36);
  });

  it("splits an ISO date", () => {
    expect(splitIsoDate("1990-07-03")).toEqual({ day: "03", month: "07", year: "1990" });
  });
});

describe("email and password", () => {
  it("normalises and validates emails", () => {
    expect(normalizeEmail("  Jane@Example.COM ")).toBe("jane@example.com");
    expect(validateEmail("jane@example.com")).toBeNull();
    expect(validateEmail("jane@")).not.toBeNull();
    expect(validateEmail("   ")).not.toBeNull();
  });

  it("enforces 8 characters and 72 bytes", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("a".repeat(72))).toBeNull();
    expect(validatePassword("a".repeat(73))).not.toBeNull();
    // 40 two-byte characters is 80 bytes although only 40 characters long.
    expect(validatePassword("é".repeat(40))).not.toBeNull();
  });

  it("counts UTF-8 bytes", () => {
    expect(utf8ByteLength("aé€😀")).toBe(1 + 2 + 3 + 4);
  });

  it("checks the confirmation", () => {
    expect(validatePasswordConfirmation("abc12345", "abc12345")).toBeNull();
    expect(validatePasswordConfirmation("abc12345", "abc1234")).not.toBeNull();
  });
});
