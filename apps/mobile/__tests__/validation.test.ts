import {
  computeAge,
  parseBirthDate,
  validateAgeRange,
  validateBio,
  validateBirthDate,
  validateEmail,
  validateFirstName,
  validatePassword
} from "../src/lib/validation";

const NOW = new Date(Date.UTC(2026, 5, 15));

describe("birth date", () => {
  it("parses valid dates and rejects impossible ones", () => {
    expect(parseBirthDate("1990-02-28")).not.toBeNull();
    expect(parseBirthDate("1990-02-30")).toBeNull();
    expect(parseBirthDate("90-02-01")).toBeNull();
    expect(parseBirthDate("not a date")).toBeNull();
  });

  it("computes age around the birthday", () => {
    const birth = parseBirthDate("2000-06-16") as Date;
    expect(computeAge(birth, NOW)).toBe(25);
    expect(computeAge(parseBirthDate("2000-06-15") as Date, NOW)).toBe(26);
  });

  it("enforces 18+", () => {
    expect(validateBirthDate("2008-06-15", NOW)).toBeNull();
    expect(validateBirthDate("2008-06-16", NOW)).toMatch(/at least 18/);
    expect(validateBirthDate("1850-01-01", NOW)).toMatch(/valid/);
    expect(validateBirthDate("garbage", NOW)).toMatch(/YYYY-MM-DD/);
  });
});

describe("form validators", () => {
  it("validates email", () => {
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("a@b")).not.toBeNull();
    expect(validateEmail(" user@example.com ")).toBeNull();
  });
  it("validates password length", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long enough")).toBeNull();
    expect(validatePassword("x".repeat(129))).not.toBeNull();
  });
  it("validates names and bio", () => {
    expect(validateFirstName("  ")).not.toBeNull();
    expect(validateFirstName("Ana")).toBeNull();
    expect(validateBio("x".repeat(501))).not.toBeNull();
    expect(validateBio("hi")).toBeNull();
  });
  it("validates age ranges", () => {
    expect(validateAgeRange(25, 30)).toBeNull();
    expect(validateAgeRange(30, 25)).not.toBeNull();
    expect(validateAgeRange(17, 30)).not.toBeNull();
    expect(validateAgeRange(20, 120)).not.toBeNull();
  });
});
