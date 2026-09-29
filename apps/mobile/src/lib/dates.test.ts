import { describe, expect, it } from "vitest";
import { computeAge, isValidCalendarDate, maskBirthDateInput, toIsoDate, validateBirthDate } from "./dates";

const today = new Date(2026, 5, 15); // 15 June 2026

describe("maskBirthDateInput", () => {
  it("inserts slashes and strips junk", () => {
    expect(maskBirthDateInput("1")).toBe("1");
    expect(maskBirthDateInput("150")).toBe("15/0");
    expect(maskBirthDateInput("15061990")).toBe("15/06/1990");
    expect(maskBirthDateInput("15/06/1990999")).toBe("15/06/1990");
    expect(maskBirthDateInput("ab1c5")).toBe("15");
  });
});

describe("isValidCalendarDate", () => {
  it("handles leap years and month lengths", () => {
    expect(isValidCalendarDate(2000, 2, 29)).toBe(true);
    expect(isValidCalendarDate(1900, 2, 29)).toBe(false);
    expect(isValidCalendarDate(2001, 2, 29)).toBe(false);
    expect(isValidCalendarDate(2001, 4, 31)).toBe(false);
    expect(isValidCalendarDate(2001, 13, 1)).toBe(false);
    expect(isValidCalendarDate(2001, 0, 1)).toBe(false);
  });
});

describe("computeAge", () => {
  it("counts a birthday only once it has happened", () => {
    expect(computeAge(2000, 6, 15, today)).toBe(26);
    expect(computeAge(2000, 6, 16, today)).toBe(25);
    expect(computeAge(2000, 7, 1, today)).toBe(25);
    expect(computeAge(2000, 1, 1, today)).toBe(26);
  });
});

describe("toIsoDate", () => {
  it("zero pads", () => {
    expect(toIsoDate(1990, 3, 7)).toBe("1990-03-07");
  });
});

describe("validateBirthDate", () => {
  it("accepts an adult and returns ISO + age", () => {
    expect(validateBirthDate("15/06/1990", today)).toEqual({ ok: true, iso: "1990-06-15", age: 36 });
  });
  it("accepts exactly 18 today", () => {
    expect(validateBirthDate("15/06/2008", today)).toMatchObject({ ok: true, age: 18 });
  });
  it("rejects one day short of 18", () => {
    const result = validateBirthDate("16/06/2008", today);
    expect(result.ok).toBe(false);
  });
  it("rejects incomplete, impossible, future and absurd dates", () => {
    expect(validateBirthDate("15/06", today).ok).toBe(false);
    expect(validateBirthDate("31/02/1990", today).ok).toBe(false);
    expect(validateBirthDate("01/01/2030", today).ok).toBe(false);
    expect(validateBirthDate("01/01/1800", today).ok).toBe(false);
  });
  it("explains the under-18 case", () => {
    const result = validateBirthDate("01/01/2015", today);
    expect(result).toEqual({ ok: false, error: "You must be at least 18 to use Alba." });
  });
});
