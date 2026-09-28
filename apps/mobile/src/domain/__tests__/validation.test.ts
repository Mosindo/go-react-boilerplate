import { isValidEmail, normalizeEmail, passwordError } from "../validation";

describe("validation", () => {
  it("normalizes email", () => {
    expect(normalizeEmail("  Ana@Example.COM ")).toBe("ana@example.com");
  });

  it("validates email shape", () => {
    expect(isValidEmail("ana@example.com")).toBe(true);
    expect(isValidEmail(" ana@example.com ")).toBe(true);
    expect(isValidEmail("ana@example")).toBe(false);
    expect(isValidEmail("ana example@x.com")).toBe(false);
  });

  it("enforces password length 8..128", () => {
    expect(passwordError("short")).not.toBeNull();
    expect(passwordError("longenough")).toBeNull();
    expect(passwordError("x".repeat(129))).not.toBeNull();
  });
});
