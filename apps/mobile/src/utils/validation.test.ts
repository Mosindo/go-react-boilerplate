import { maskBirthdate, normalizeEmail, parseBirthdate, validateEmail, validatePassword, isoToFrench } from "./validation";

describe("validation", () => {
  it("normalizes and validates emails", () => {
    expect(normalizeEmail("  Alice@Example.COM ")).toBe("alice@example.com");
    expect(validateEmail("alice@example.com")).toBeNull();
    expect(validateEmail("alice@")).not.toBeNull();
  });

  it("applies the password policy", () => {
    expect(validatePassword("Password123")).toBeNull();
    expect(validatePassword("short1")).not.toBeNull();
    expect(validatePassword("onlyletters")).not.toBeNull();
    expect(validatePassword("a1".repeat(37))).toBe("72 caractères maximum.");
  });

  it("parses birthdates and enforces the minimum age", () => {
    const now = new Date(2026, 10, 25);
    expect(parseBirthdate("02/04/1995", now)).toEqual({ iso: "1995-04-02", age: 31 });
    expect(parseBirthdate("26/11/2008", now)).toMatch(/18 ans/);
    expect(parseBirthdate("25/11/2008", now)).toEqual({ iso: "2008-11-25", age: 18 });
    expect(parseBirthdate("31/02/1995", now)).toBe("Cette date n'existe pas.");
    expect(parseBirthdate("1995-04-02", now)).toMatch(/JJ\/MM\/AAAA/);
  });

  it("masks and converts birthdates", () => {
    expect(maskBirthdate("0204")).toBe("02/04");
    expect(maskBirthdate("02041995")).toBe("02/04/1995");
    expect(maskBirthdate("02/04/19951")).toBe("02/04/1995");
    expect(isoToFrench("1995-04-02")).toBe("02/04/1995");
  });
});
