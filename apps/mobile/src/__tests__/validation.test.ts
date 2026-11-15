import { ageOn, checkBirthdate, isValidEmail, isValidFirstName, passwordProblem } from "../lib/validation";

describe("validation", () => {
  const now = new Date(2026, 5, 15); // 15 June 2026

  it("computes age around birthdays", () => {
    expect(ageOn(new Date(2008, 5, 15), now)).toBe(18);
    expect(ageOn(new Date(2008, 5, 16), now)).toBe(17);
  });

  it("validates birthdates and enforces 18+", () => {
    expect(checkBirthdate("12", "04", "1995", now)).toEqual({ ok: true, iso: "1995-04-12", age: 31 });
    expect(checkBirthdate("16", "06", "2008", now)).toMatchObject({ ok: false });
    expect(checkBirthdate("30", "02", "1995", now)).toMatchObject({ ok: false, error: "Cette date n'existe pas." });
    expect(checkBirthdate("1", "1", "95", now)).toMatchObject({ ok: false });
  });

  it("mirrors the server password policy", () => {
    expect(passwordProblem("short1")).not.toBeNull();
    expect(passwordProblem("allletters")).not.toBeNull();
    expect(passwordProblem("12345678")).not.toBeNull();
    expect(passwordProblem("Secret123")).toBeNull();
  });

  it("validates emails and first names", () => {
    expect(isValidEmail(" alice@example.com ")).toBe(true);
    expect(isValidEmail("alice@example")).toBe(false);
    expect(isValidFirstName("Anne-Marie")).toBe(true);
    expect(isValidFirstName("Zoé")).toBe(true);
    expect(isValidFirstName("B0b")).toBe(false);
    expect(isValidFirstName("<b>")).toBe(false);
    expect(isValidFirstName("   ")).toBe(false);
  });
});
