import { describe, expect, it } from "vitest";
import { isValidEmail, isValidFirstName, isValidPassword } from "./validation";

describe("validation", () => {
  it("validates emails", () => {
    expect(isValidEmail(" a@b.fr ")).toBe(true);
    for (const bad of ["", "a@b", "a b@c.fr", "@c.fr", "a@@c.fr"])
      expect(isValidEmail(bad)).toBe(false);
  });
  it("validates passwords", () => {
    expect(isValidPassword("12345678")).toBe(true);
    expect(isValidPassword("1234567")).toBe(false);
    expect(isValidPassword("x".repeat(73))).toBe(false);
  });
  it("validates first names", () => {
    expect(isValidFirstName("Anne-Sophie")).toBe(true);
    expect(isValidFirstName("Zoë")).toBe(true);
    for (const bad of ["", "  ", "R2D2", "<b>", "-a"]) expect(isValidFirstName(bad)).toBe(false);
  });
});
