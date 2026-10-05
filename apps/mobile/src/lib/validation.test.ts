import { describe, expect, it } from "vitest";
import { isValidEmail, isValidFirstName, normalizeEmail, passwordProblem } from "./validation";

describe("email", () => {
  it("normalises case and whitespace", () => {
    expect(normalizeEmail("  Foo@Example.COM ")).toBe("foo@example.com");
  });

  it.each(["a@b.co", "first.last+tag@sub.example.org", " padded@example.com "])("accepts %s", (email) => {
    expect(isValidEmail(email)).toBe(true);
  });

  it.each(["", "plain", "a@b", "a b@c.de", "<a@b.cd>", "a@b.", "@b.cd", "a@b.cd,e@f.gh", "a@@b.cd"])("rejects %j", (email) => {
    expect(isValidEmail(email)).toBe(false);
  });
});

describe("password", () => {
  it("requires 8 characters", () => {
    expect(passwordProblem("1234567")).toMatch(/8 caractères/);
    expect(passwordProblem("12345678")).toBeNull();
  });

  it("counts bytes, because bcrypt stops at 72", () => {
    expect(passwordProblem("a".repeat(72))).toBeNull();
    expect(passwordProblem("a".repeat(73))).toMatch(/trop long/);
    expect(passwordProblem("é".repeat(40))).toMatch(/trop long/); // 80 bytes
  });
});

describe("first name", () => {
  it.each(["Léa", "Jean-Luc", "O'Neil", "Anne Marie", "Zoë"])("accepts %s", (name) => expect(isValidFirstName(name)).toBe(true));
  it.each(["", "   ", "R2D2", "<b>", "Zoé😀", "a".repeat(41)])("rejects %j", (name) => expect(isValidFirstName(name)).toBe(false));
});
