import {
  buildBirthDate,
  calculateAge,
  normalizeEmail,
  splitBirthDate,
  validateAgeRange,
  validateBirthDate,
  validateEmail,
  validateFirstName,
  validatePassword
} from "../src/utils/validation";

const NOW = new Date(2026, 5, 15); // 15 June 2026 (local)

describe("calculateAge", () => {
  it("counts whole years, not yet reached birthday", () => {
    expect(calculateAge("2000-06-16", NOW)).toBe(25);
  });
  it("counts the birthday itself", () => {
    expect(calculateAge("2000-06-15", NOW)).toBe(26);
  });
  it("returns null for garbage", () => {
    expect(calculateAge("15/06/2000", NOW)).toBeNull();
  });
});

describe("validateBirthDate (18+)", () => {
  it("rejects someone turning 18 tomorrow", () => {
    expect(validateBirthDate("2008-06-16", NOW)).not.toBeNull();
  });
  it("accepts someone turning 18 today", () => {
    expect(validateBirthDate("2008-06-15", NOW)).toBeNull();
  });
  it("accepts adults", () => {
    expect(validateBirthDate("1990-01-01", NOW)).toBeNull();
  });
  it("rejects missing, future and absurdly old dates", () => {
    expect(validateBirthDate(null, NOW)).not.toBeNull();
    expect(validateBirthDate("2030-01-01", NOW)).not.toBeNull();
    expect(validateBirthDate("1850-01-01", NOW)).not.toBeNull();
  });
});

describe("buildBirthDate", () => {
  it("pads and joins valid parts", () => {
    expect(buildBirthDate("5", "3", "1994")).toBe("1994-03-05");
  });
  it("rejects impossible calendar dates", () => {
    expect(buildBirthDate("31", "02", "2000")).toBeNull();
    expect(buildBirthDate("29", "02", "2001")).toBeNull();
    expect(buildBirthDate("29", "02", "2000")).toBe("2000-02-29");
  });
  it("rejects non numeric or short years", () => {
    expect(buildBirthDate("aa", "01", "2000")).toBeNull();
    expect(buildBirthDate("01", "01", "99")).toBeNull();
  });
  it("round trips with splitBirthDate", () => {
    expect(splitBirthDate("1994-03-05")).toEqual({ day: "05", month: "03", year: "1994" });
    expect(splitBirthDate("nope")).toEqual({ day: "", month: "", year: "" });
  });
});

describe("form validation", () => {
  it("normalizes and validates email", () => {
    expect(normalizeEmail("  Foo@Example.COM ")).toBe("foo@example.com");
    expect(validateEmail("foo@example.com")).toBeNull();
    expect(validateEmail("foo@")).not.toBeNull();
    expect(validateEmail("   ")).not.toBeNull();
  });
  it("enforces password length 8..128", () => {
    expect(validatePassword("1234567")).not.toBeNull();
    expect(validatePassword("12345678")).toBeNull();
    expect(validatePassword("x".repeat(129))).not.toBeNull();
  });
  it("enforces first name 1..40 after trimming", () => {
    expect(validateFirstName("   ")).not.toBeNull();
    expect(validateFirstName(" Ana ")).toBeNull();
    expect(validateFirstName("x".repeat(41))).not.toBeNull();
  });
  it("validates age range 18..99 with min <= max", () => {
    expect(validateAgeRange(18, 99)).toBeNull();
    expect(validateAgeRange(17, 30)).not.toBeNull();
    expect(validateAgeRange(40, 30)).not.toBeNull();
    expect(validateAgeRange(30, 100)).not.toBeNull();
  });
});
