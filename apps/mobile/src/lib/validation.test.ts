import { describe, expect, it } from "vitest";
import { checkPassword, cleanName, isValidEmail, isValidFirstName, normalizeEmail } from "./validation";

describe("email", () => {
  it("normalizes and validates", () => {
    expect(normalizeEmail("  Alice@Example.COM ")).toBe("alice@example.com");
    expect(isValidEmail("alice@example.com")).toBe(true);
    expect(isValidEmail(" alice@example.com ")).toBe(true);
    for (const bad of ["", "alice", "alice@", "@example.com", "alice@example", "a b@example.com", "a@b.c"]) {
      expect(isValidEmail(bad)).toBe(false);
    }
  });
});

describe("password", () => {
  it("requires 8 characters and at most 72 bytes (bcrypt limit)", () => {
    expect(checkPassword("1234567")).toBe("short");
    expect(checkPassword("12345678")).toBeNull();
    expect(checkPassword("a".repeat(72))).toBeNull();
    expect(checkPassword("a".repeat(73))).toBe("long");
    expect(checkPassword("é".repeat(37))).toBe("long"); // 74 bytes
    expect(checkPassword("é".repeat(36))).toBeNull(); // 72 bytes
  });
});

describe("first name", () => {
  it("collapses whitespace and bounds the length", () => {
    expect(cleanName("  Jean   Pierre ")).toBe("Jean Pierre");
    expect(isValidFirstName("   ")).toBe(false);
    expect(isValidFirstName("É")).toBe(true);
    expect(isValidFirstName("a".repeat(41))).toBe(false);
  });
});
