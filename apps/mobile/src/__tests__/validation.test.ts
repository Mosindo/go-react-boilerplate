import {
  ageFromBirthDate,
  toBirthDate,
  validateAgeRange,
  validateBirthDate,
  validateEmail,
  validateMessageBody,
  validatePassword
} from "../lib/validation";

const NOW = new Date(Date.UTC(2026, 5, 15));

describe("validation", () => {
  it("validates emails", () => {
    expect(validateEmail("  ")).not.toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validateEmail(" Me@Example.com ")).toBeNull();
  });

  it("validates passwords", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long enough")).toBeNull();
    expect(validatePassword("x".repeat(129))).not.toBeNull();
  });

  it("builds real calendar dates only", () => {
    expect(toBirthDate("1990", "2", "30")).toBeNull();
    expect(toBirthDate("90", "2", "3")).toBeNull();
    expect(toBirthDate("1990", "2", "3")).toBe("1990-02-03");
  });

  it("computes age around the birthday", () => {
    expect(ageFromBirthDate("2000-06-15", NOW)).toBe(26);
    expect(ageFromBirthDate("2000-06-16", NOW)).toBe(25);
  });

  it("enforces the 18+ rule", () => {
    expect(validateBirthDate("2008-06-16", NOW)).toMatch(/at least 18/);
    expect(validateBirthDate("2008-06-15", NOW)).toBeNull();
    expect(validateBirthDate(null, NOW)).not.toBeNull();
    expect(validateBirthDate("1900-01-01", NOW)).not.toBeNull();
  });

  it("validates age ranges and messages", () => {
    expect(validateAgeRange(25, 20)).not.toBeNull();
    expect(validateAgeRange(17, 30)).not.toBeNull();
    expect(validateAgeRange(20, 30)).toBeNull();
    expect(validateMessageBody("   ")).not.toBeNull();
    expect(validateMessageBody("hi")).toBeNull();
  });
});
