import { ageFromBirthDate, parseBirthDate, splitBirthDate } from "../age";

const now = new Date(2026, 5, 15); // 15 June 2026

describe("ageFromBirthDate", () => {
  it("counts a birthday that already happened this year", () => {
    expect(ageFromBirthDate("2000-06-15", now)).toBe(26);
    expect(ageFromBirthDate("2000-01-01", now)).toBe(26);
  });

  it("does not count a birthday that is still ahead", () => {
    expect(ageFromBirthDate("2000-06-16", now)).toBe(25);
    expect(ageFromBirthDate("2000-12-31", now)).toBe(25);
  });
});

describe("parseBirthDate", () => {
  it("accepts an adult and returns an ISO date", () => {
    expect(parseBirthDate("5", "3", "1990", now)).toEqual({ ok: true, iso: "1990-03-05", age: 36 });
  });

  it("accepts someone turning 18 today", () => {
    const result = parseBirthDate("15", "06", "2008", now);
    expect(result).toEqual({ ok: true, iso: "2008-06-15", age: 18 });
  });

  it("rejects someone turning 18 tomorrow", () => {
    const result = parseBirthDate("16", "06", "2008", now);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toMatch(/at least 18/);
    }
  });

  it("rejects dates that do not exist", () => {
    expect(parseBirthDate("31", "02", "1995", now)).toEqual({ ok: false, error: "That date does not exist." });
    expect(parseBirthDate("29", "02", "2001", now).ok).toBe(false);
    expect(parseBirthDate("29", "02", "2000", now).ok).toBe(true);
  });

  it("rejects non numeric, short year and out of range month input", () => {
    expect(parseBirthDate("ab", "01", "1990", now).ok).toBe(false);
    expect(parseBirthDate("1", "1", "90", now).ok).toBe(false);
    expect(parseBirthDate("1", "13", "1990", now).ok).toBe(false);
    expect(parseBirthDate("", "", "", now).ok).toBe(false);
  });

  it("rejects future dates", () => {
    const result = parseBirthDate("1", "1", "2030", now);
    expect(result).toEqual({ ok: false, error: "Birth date cannot be in the future." });
  });
});

describe("splitBirthDate", () => {
  it("splits an ISO date into text parts", () => {
    expect(splitBirthDate("1990-03-05")).toEqual({ day: "05", month: "03", year: "1990" });
  });
});
