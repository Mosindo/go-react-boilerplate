import { describe, expect, it } from "vitest";
import { ageOn, formatDayLabel, formatRelative, isSameDay, parseBirthDate, splitIsoDate } from "./dates";

const now = new Date(2026, 5, 15, 12, 0, 0); // 15 June 2026

describe("parseBirthDate", () => {
  it("accepts a valid adult date and reports the age", () => {
    expect(parseBirthDate("15", "6", "1990", now)).toEqual({ ok: true, iso: "1990-06-15", age: 36 });
  });

  it("accepts someone turning 18 today but not tomorrow", () => {
    expect(parseBirthDate("15", "6", "2008", now)).toMatchObject({ ok: true, age: 18 });
    expect(parseBirthDate("16", "6", "2008", now)).toMatchObject({ ok: false });
  });

  it("rejects minors with an explicit message", () => {
    const result = parseBirthDate("1", "1", "2015", now);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("18 ans");
  });

  it.each([
    ["31", "2", "1990"], // 31 February
    ["29", "2", "2023"], // not a leap year
    ["0", "5", "1990"],
    ["10", "13", "1990"],
    ["aa", "5", "1990"],
    ["10", "5", "90"],
    ["", "", ""]
  ])("rejects impossible or malformed date %s/%s/%s", (d, m, y) => {
    expect(parseBirthDate(d, m, y, now).ok).toBe(false);
  });

  it("rejects implausibly old dates", () => {
    expect(parseBirthDate("1", "1", "1890", now).ok).toBe(false);
  });

  it("accepts a leap day", () => {
    expect(parseBirthDate("29", "2", "2000", now)).toMatchObject({ ok: true, iso: "2000-02-29" });
  });
});

describe("helpers", () => {
  it("computes age on a given day", () => {
    expect(ageOn("2000-06-16", now)).toBe(25);
    expect(ageOn("2000-06-15", now)).toBe(26);
  });

  it("splits an ISO date for the form", () => {
    expect(splitIsoDate("1990-06-05")).toEqual({ day: "5", month: "6", year: "1990" });
  });

  it("labels days relative to now", () => {
    expect(formatDayLabel(new Date(2026, 5, 15, 8).toISOString(), now)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 5, 14, 23).toISOString(), now)).toBe("Hier");
    expect(formatRelative(new Date(2026, 5, 14, 23).toISOString(), now)).toBe("Hier");
    expect(formatRelative(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toMatch(/09:05|9:05/);
    expect(formatRelative(new Date(2026, 0, 3).toISOString(), now)).toContain("janvier");
  });

  it("compares calendar days, not 24-hour windows", () => {
    expect(isSameDay(new Date(2026, 5, 15, 0, 1).toISOString(), new Date(2026, 5, 15, 23, 59).toISOString())).toBe(true);
    expect(isSameDay(new Date(2026, 5, 15, 23, 59).toISOString(), new Date(2026, 5, 16, 0, 1).toISOString())).toBe(false);
  });
});
