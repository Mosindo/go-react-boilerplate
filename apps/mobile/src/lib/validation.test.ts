import { normalizeEmail, validateEmail, validateFirstName, validatePassword } from "./validation";

describe("validation", () => {
  it("normalizes and validates emails", () => {
    expect(normalizeEmail("  Ana@Example.COM ")).toBe("ana@example.com");
    expect(validateEmail("ana@example.com")).toBeNull();
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("not-an-email")).not.toBeNull();
    expect(validateEmail("a@b")).not.toBeNull();
  });

  it("mirrors the server password rule", () => {
    expect(validatePassword("Password123")).toBeNull();
    expect(validatePassword("short1")).not.toBeNull();
    expect(validatePassword("onlyletters")).not.toBeNull();
    expect(validatePassword("12345678")).not.toBeNull();
    expect(validatePassword("a1".repeat(40))).not.toBeNull();
  });

  it("accepts names with accents and hyphens only", () => {
    expect(validateFirstName("Anne-Laure")).toBeNull();
    expect(validateFirstName("Zoé")).toBeNull();
    expect(validateFirstName("<b>")).not.toBeNull();
    expect(validateFirstName("Bob3")).not.toBeNull();
    expect(validateFirstName("  ")).not.toBeNull();
  });
});
