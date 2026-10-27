import {
  ageFromBirthDate,
  describeDistance,
  formatBirthDateInput,
  formatTime
} from "../lib/format";
import { validateEmail, validatePassword, validateProfileDraft } from "../lib/validation";

describe("format", () => {
  it("formats birth date input progressively", () => {
    expect(formatBirthDateInput("1993")).toBe("1993");
    expect(formatBirthDateInput("199304")).toBe("1993-04");
    expect(formatBirthDateInput("19930412")).toBe("1993-04-12");
    expect(formatBirthDateInput("1993-04-12xyz99")).toBe("1993-04-12");
  });

  it("computes age and rejects impossible dates", () => {
    const now = new Date(Date.UTC(2026, 5, 15));
    expect(ageFromBirthDate("2008-06-15", now)).toBe(18);
    expect(ageFromBirthDate("2008-06-16", now)).toBe(17);
    expect(ageFromBirthDate("2001-02-30", now)).toBeNull();
    expect(ageFromBirthDate("nope", now)).toBeNull();
  });

  it("describes distance and times", () => {
    expect(describeDistance(null)).toBeNull();
    expect(describeDistance(15)).toBe("15 km away");
    expect(formatTime("garbage")).toBe("");
    const now = new Date("2026-06-15T12:00:00");
    expect(formatTime("2026-06-15T09:30:00", now)).toMatch(/\d/);
    expect(formatTime("2025-01-02T09:30:00", now)).toMatch(/Jan|2/);
  });
});

describe("validation", () => {
  const now = new Date(Date.UTC(2026, 5, 15));
  const valid = {
    firstName: "Zoé",
    birthDate: "1995-03-10",
    gender: "woman" as const,
    bio: "",
    city: "Lyon"
  };

  it("accepts a valid profile", () => {
    expect(validateProfileDraft(valid, now)).toEqual({});
  });

  it("rejects minors, bad names and missing gender", () => {
    expect(validateProfileDraft({ ...valid, birthDate: "2010-01-01" }, now).birthDate).toMatch(
      /18/
    );
    expect(validateProfileDraft({ ...valid, firstName: "R2D2" }, now).firstName).toBeDefined();
    expect(validateProfileDraft({ ...valid, firstName: "  " }, now).firstName).toBeDefined();
    expect(validateProfileDraft({ ...valid, gender: null }, now).gender).toBeDefined();
    expect(validateProfileDraft({ ...valid, bio: "x".repeat(501) }, now).bio).toBeDefined();
  });

  it("validates credentials", () => {
    expect(validateEmail("a@b.co")).toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("x".repeat(73))).not.toBeNull();
  });
});
