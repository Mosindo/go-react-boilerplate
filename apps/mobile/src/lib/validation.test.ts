import { describe, expect, it } from "vitest";
import {
  hasErrors,
  normalizeEmail,
  stepAgeMax,
  stepAgeMin,
  stepDistance,
  toggleInList,
  validateEmail,
  validateLoginPassword,
  validateNewPassword,
  validatePassword,
  validatePreferences,
  validateProfileForm
} from "./validation";

describe("email", () => {
  it("trims and lowercases", () => {
    expect(normalizeEmail("  Sam@Example.COM ")).toBe("sam@example.com");
  });
  it("validates shape", () => {
    expect(validateEmail("sam@example.com")).toBeNull();
    expect(validateEmail("  ")).not.toBeNull();
    expect(validateEmail("sam@")).not.toBeNull();
    expect(validateEmail("sam example@x.io")).not.toBeNull();
    expect(validateEmail("a@b")).not.toBeNull();
  });
});

describe("password", () => {
  it("enforces 8-128", () => {
    expect(validatePassword("1234567")).not.toBeNull();
    expect(validatePassword("12345678")).toBeNull();
    expect(validatePassword("x".repeat(128))).toBeNull();
    expect(validatePassword("x".repeat(129))).not.toBeNull();
    expect(validatePassword("")).not.toBeNull();
  });
  it("login only needs presence", () => {
    expect(validateLoginPassword("a")).toBeNull();
    expect(validateLoginPassword("")).not.toBeNull();
  });
  it("new password must differ from current", () => {
    expect(validateNewPassword("password1", "password1")).not.toBeNull();
    expect(validateNewPassword("password2", "password1")).toBeNull();
    expect(validateNewPassword("short", "password1")).not.toBeNull();
  });
});

describe("validateProfileForm", () => {
  const valid = { firstName: " Sam ", gender: "man" as const, bio: "", city: "", interests: [] };
  it("accepts a minimal profile", () => {
    expect(hasErrors(validateProfileForm(valid))).toBe(false);
  });
  it("flags name, gender, bio, city, interests", () => {
    const errors = validateProfileForm({
      firstName: "   ",
      gender: null,
      bio: "x".repeat(501),
      city: "c".repeat(81),
      interests: Array.from({ length: 11 }, (_, i) => `i${i}`)
    });
    expect(Object.keys(errors).sort()).toEqual(["bio", "city", "firstName", "gender", "interests"]);
  });
  it("counts the trimmed name length", () => {
    expect(validateProfileForm({ ...valid, firstName: "a".repeat(41) }).firstName).toBeDefined();
    expect(validateProfileForm({ ...valid, firstName: "a".repeat(40) }).firstName).toBeUndefined();
  });
});

describe("validatePreferences", () => {
  const ok = { interestedIn: ["woman" as const], ageMin: 25, ageMax: 35, maxDistanceKm: 50 };
  it("accepts valid", () => {
    expect(hasErrors(validatePreferences(ok))).toBe(false);
  });
  it("rejects bad values", () => {
    expect(validatePreferences({ ...ok, interestedIn: [] }).interestedIn).toBeDefined();
    expect(validatePreferences({ ...ok, ageMin: 17 }).ageMin).toBeDefined();
    expect(validatePreferences({ ...ok, ageMax: 20 }).ageMax).toBeDefined();
    expect(validatePreferences({ ...ok, ageMax: 100 }).ageMax).toBeDefined();
    expect(validatePreferences({ ...ok, maxDistanceKm: 0 }).maxDistanceKm).toBeDefined();
    expect(validatePreferences({ ...ok, maxDistanceKm: 501 }).maxDistanceKm).toBeDefined();
  });
});

describe("steppers", () => {
  const base = { interestedIn: ["man" as const], ageMin: 30, ageMax: 32, maxDistanceKm: 50 };
  it("keeps the age range coherent", () => {
    expect(stepAgeMin(base, 5)).toMatchObject({ ageMin: 35, ageMax: 35 });
    expect(stepAgeMax(base, -5)).toMatchObject({ ageMin: 27, ageMax: 27 });
    expect(stepAgeMin(base, -100).ageMin).toBe(18);
    expect(stepAgeMax(base, 100).ageMax).toBe(99);
  });
  it("steps distance sensibly", () => {
    expect(stepDistance(5, 1)).toBe(6);
    expect(stepDistance(9, 1)).toBe(10);
    expect(stepDistance(10, 1)).toBe(15);
    expect(stepDistance(12, 1)).toBe(15);
    expect(stepDistance(15, -1)).toBe(10);
    expect(stepDistance(10, -1)).toBe(9);
    expect(stepDistance(12, -1)).toBe(10);
    expect(stepDistance(1, -1)).toBe(1);
    expect(stepDistance(500, 1)).toBe(500);
  });
});

describe("toggleInList", () => {
  it("adds, removes and respects max", () => {
    expect(toggleInList(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleInList(["a", "b"], "a")).toEqual(["b"]);
    expect(toggleInList(["a", "b"], "c", 2)).toEqual(["a", "b"]);
    expect(toggleInList(["a", "b"], "a", 2)).toEqual(["b"]);
  });
});
