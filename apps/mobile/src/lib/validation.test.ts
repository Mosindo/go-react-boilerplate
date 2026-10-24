import { describe, expect, it } from "vitest";
import { errorMessage, validateEmail, validateNewPassword } from "./validation";

describe("validateEmail", () => {
  it("accepts normal addresses (trimmed) and rejects the rest", () => {
    expect(validateEmail("  ana@example.org ")).toBeNull();
    for (const bad of ["", "ana", "ana@", "@example.org", "ana@example", "a b@c.d"]) {
      expect(validateEmail(bad)).not.toBeNull();
    }
  });
});

describe("validateNewPassword", () => {
  it("enforces 8..72 bytes", () => {
    expect(validateNewPassword("1234567")).not.toBeNull();
    expect(validateNewPassword("12345678")).toBeNull();
    expect(validateNewPassword("a".repeat(73))).not.toBeNull();
    expect(validateNewPassword("é".repeat(37))).not.toBeNull(); // 74 bytes
  });
});

describe("errorMessage", () => {
  it("falls back for non-errors", () => {
    expect(errorMessage(new Error("boom"))).toBe("boom");
    expect(errorMessage("x", "fallback")).toBe("fallback");
  });
});
