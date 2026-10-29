import { validateEmail, validatePassword } from "./validation";

describe("validation", () => {
  it("validates emails", () => {
    expect(validateEmail("  Jane@Example.com ")).toBeNull();
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("jane@")).not.toBeNull();
    expect(validateEmail("jane doe@example.com")).not.toBeNull();
  });

  it("validates passwords like the server does", () => {
    expect(validatePassword("12345678")).toBeNull();
    expect(validatePassword("1234567")).not.toBeNull();
    expect(validatePassword("        ")).not.toBeNull();
    expect(validatePassword("a".repeat(73))).not.toBeNull();
    expect(validatePassword("é".repeat(37))).not.toBeNull(); // 74 bytes
  });
});
