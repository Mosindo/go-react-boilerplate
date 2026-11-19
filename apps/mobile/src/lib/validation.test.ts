import {
  ageFrom,
  birthdateProblem,
  hasLetter,
  isValidEmail,
  isoToFrenchDate,
  maskDateInput,
  normalizeEmail,
  parseFrenchDate,
  passwordProblem
} from "./validation";

describe("validation", () => {
  const now = new Date(2026, 5, 15);

  it("normalises and validates emails", () => {
    expect(normalizeEmail("  Alice@Example.COM ")).toBe("alice@example.com");
    expect(isValidEmail("alice@example.com")).toBe(true);
    expect(isValidEmail("alice@example")).toBe(false);
    expect(isValidEmail("not an email")).toBe(false);
  });

  it("mirrors the server password rule", () => {
    expect(passwordProblem("short1")).not.toBeNull();
    expect(passwordProblem("abcdefgh")).not.toBeNull();
    expect(passwordProblem("12345678")).not.toBeNull();
    expect(passwordProblem("s3cret-password")).toBeNull();
  });

  it("parses French dates strictly", () => {
    expect(parseFrenchDate("02/03/1994")).toBe("1994-03-02");
    expect(parseFrenchDate("31/02/1994")).toBeNull();
    expect(parseFrenchDate("1994-03-02")).toBeNull();
    expect(isoToFrenchDate("1994-03-02")).toBe("02/03/1994");
  });

  it("enforces the minimum age", () => {
    expect(birthdateProblem("16/06/2008", now)).toMatch(/18 ans/);
    expect(birthdateProblem("15/06/2008", now)).toBeNull();
    expect(birthdateProblem("01/01/1900", now)).not.toBeNull();
    expect(ageFrom(new Date(1990, 11, 31), now)).toBe(35);
  });

  it("masks date input while typing", () => {
    expect(maskDateInput("0")).toBe("0");
    expect(maskDateInput("0203")).toBe("02/03");
    expect(maskDateInput("02031994999")).toBe("02/03/1994");
  });

  it("detects letters in any script", () => {
    expect(hasLetter("Léa")).toBe(true);
    expect(hasLetter("明")).toBe(true);
    expect(hasLetter("1234")).toBe(false);
    expect(hasLetter("   ")).toBe(false);
  });
});
