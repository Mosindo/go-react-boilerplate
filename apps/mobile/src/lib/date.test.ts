import { describe, expect, it } from "vitest";
import { ageAt, formatShortTime, parseBirthDate, splitIsoDate } from "./date";

const now = new Date(2026, 5, 15, 12, 0, 0); // 15 June 2026

describe("parseBirthDate", () => {
  it("accepts an adult and returns the ISO date and age", () => {
    expect(parseBirthDate("2", "3", "1995", now)).toEqual({ ok: true, iso: "1995-03-02", age: 31 });
  });

  it("accepts someone turning 18 today and rejects the day before", () => {
    expect(parseBirthDate("15", "6", "2008", now)).toMatchObject({ ok: true, age: 18 });
    expect(parseBirthDate("16", "6", "2008", now)).toMatchObject({ ok: false });
  });

  it("rejects impossible and malformed dates", () => {
    for (const [d, m, y] of [
      ["31", "2", "1990"],
      ["0", "1", "1990"],
      ["10", "13", "1990"],
      ["a", "1", "1990"],
      ["1", "1", "90"],
      ["", "", ""],
    ] as const) {
      expect(parseBirthDate(d, m, y, now).ok).toBe(false);
    }
  });

  it("accepts a leap day and rejects an absurd age", () => {
    expect(parseBirthDate("29", "2", "2000", now)).toMatchObject({ ok: true });
    expect(parseBirthDate("1", "1", "1890", now).ok).toBe(false);
  });
});

describe("ageAt", () => {
  it("handles birthdays that have not happened yet this year", () => {
    expect(ageAt({ year: 2000, month: 12, day: 31 }, now)).toBe(25);
  });
});

describe("splitIsoDate", () => {
  it("splits an ISO date into form fields", () => {
    expect(splitIsoDate("1995-03-02")).toEqual({ day: "02", month: "03", year: "1995" });
  });
});

describe("formatShortTime", () => {
  it("shows the clock time today, 'hier' yesterday and dd/mm otherwise", () => {
    expect(formatShortTime(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatShortTime(new Date(2026, 5, 14, 23, 59).toISOString(), now)).toBe("hier");
    expect(formatShortTime(new Date(2026, 4, 2, 10, 0).toISOString(), now)).toBe("02/05");
    expect(formatShortTime("garbage", now)).toBe("");
  });
});
