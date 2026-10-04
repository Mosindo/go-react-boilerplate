import { clampAgeRange, clampDistance, normalizeEmail, validateEmail, validatePassword } from "../validation";

describe("email", () => {
  it("normalizes and validates", () => {
    expect(normalizeEmail("  Foo@Bar.COM ")).toBe("foo@bar.com");
    expect(validateEmail("foo@bar.com")).toBeNull();
    expect(validateEmail("foo@bar")).not.toBeNull();
    expect(validateEmail("   ")).not.toBeNull();
  });
});

describe("password", () => {
  it("enforces 8 characters and 72 bytes", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("a".repeat(72))).toBeNull();
    expect(validatePassword("a".repeat(73))).not.toBeNull();
    // 40 emoji = 160 bytes even though only 40 code points
    expect(validatePassword("😀".repeat(20))).not.toBeNull();
  });
});

describe("ranges", () => {
  it("keeps min <= max inside 18..99, moving the other bound", () => {
    expect(clampAgeRange({ minAge: 40, maxAge: 30 }, "min")).toEqual({ minAge: 40, maxAge: 40 });
    expect(clampAgeRange({ minAge: 40, maxAge: 30 }, "max")).toEqual({ minAge: 30, maxAge: 30 });
    expect(clampAgeRange({ minAge: 5, maxAge: 200 }, "min")).toEqual({ minAge: 18, maxAge: 99 });
  });
  it("clamps distance to 1..500", () => {
    expect(clampDistance(0)).toBe(1);
    expect(clampDistance(9999)).toBe(500);
    expect(clampDistance(42.4)).toBe(42);
  });
});
