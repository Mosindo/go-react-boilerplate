import { ageFromBirthdate, isValidEmail, normalizeEmail, parseBirthdate, passwordIssue, roundCoordinate } from "./validation";

const NOW = new Date(2026, 5, 15); // 15 June 2026

describe("email", () => {
  it("normalises and validates", () => {
    expect(normalizeEmail("  Bob@Example.ORG ")).toBe("bob@example.org");
    expect(isValidEmail("a@b.co")).toBe(true);
    expect(isValidEmail("a@b")).toBe(false);
    expect(isValidEmail("a b@c.de")).toBe(false);
    expect(isValidEmail("")).toBe(false);
  });
});

describe("password", () => {
  it("enforces the length rules", () => {
    expect(passwordIssue("short")).not.toBeNull();
    expect(passwordIssue("long-enough-1")).toBeNull();
    expect(passwordIssue("a".repeat(73))).not.toBeNull();
  });
});

describe("age", () => {
  it("counts whole years", () => {
    expect(ageFromBirthdate("2008-06-15", NOW)).toBe(18);
    expect(ageFromBirthdate("2008-06-16", NOW)).toBe(17);
    expect(ageFromBirthdate("1990-12-31", NOW)).toBe(35);
  });
});

describe("parseBirthdate", () => {
  it("accepts adults", () => {
    expect(parseBirthdate("15", "06", "2008", NOW)).toEqual({ ok: true, iso: "2008-06-15", age: 18 });
  });
  it("rejects minors the day before their 18th birthday", () => {
    const result = parseBirthdate("16", "06", "2008", NOW);
    expect(result.ok).toBe(false);
  });
  it("rejects impossible and malformed dates", () => {
    expect(parseBirthdate("31", "02", "1990", NOW).ok).toBe(false);
    expect(parseBirthdate("xx", "02", "1990", NOW).ok).toBe(false);
    expect(parseBirthdate("10", "10", "90", NOW).ok).toBe(false);
    expect(parseBirthdate("10", "10", "1900", NOW).ok).toBe(false);
  });
});

describe("roundCoordinate", () => {
  it("keeps roughly one kilometre of precision", () => {
    expect(roundCoordinate(48.85661)).toBe(48.86);
    expect(roundCoordinate(-73.5673)).toBe(-73.57);
  });
});
