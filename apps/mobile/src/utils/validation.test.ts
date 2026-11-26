import { ageOn, firstNameProblem, isValidEmail, normalizeEmail, parseBirthdate, passwordProblem } from "./validation";

describe("validation", () => {
  const today = new Date(2026, 5, 15);

  it("computes ages on birthdays", () => {
    expect(ageOn(new Date(2008, 5, 15), today)).toBe(18);
    expect(ageOn(new Date(2008, 5, 16), today)).toBe(17);
  });

  it("only accepts adults and real dates", () => {
    expect(parseBirthdate("15", "06", "2008", today)).toEqual({ ok: true, iso: "2008-06-15", age: 18 });
    expect(parseBirthdate("16", "06", "2008", today).ok).toBe(false);
    expect(parseBirthdate("31", "02", "1990", today).ok).toBe(false);
    expect(parseBirthdate("1", "1", "90", today).ok).toBe(false);
    expect(parseBirthdate("01", "01", "1900", today).ok).toBe(false);
  });

  it("normalises and validates e-mails", () => {
    expect(normalizeEmail("  Léa@Example.FR ")).toBe("léa@example.fr");
    expect(isValidEmail(" a@b.co ")).toBe(true);
    expect(isValidEmail("a@b")).toBe(false);
  });

  it("enforces the password policy", () => {
    expect(passwordProblem("short1")).not.toBeNull();
    expect(passwordProblem("lettersonly")).not.toBeNull();
    expect(passwordProblem("Password1")).toBeNull();
  });

  it("validates first names", () => {
    expect(firstNameProblem("Anne-Marie d’Arc")).toBeNull();
    expect(firstNameProblem("   ")).not.toBeNull();
    expect(firstNameProblem("R2D2")).not.toBeNull();
    expect(firstNameProblem("<b>")).not.toBeNull();
  });
});
