import { describe, expect, it } from "vitest";
import { ageOn, formatListTime, parseBirthDate, splitIsoDate } from "./dates";

const now = new Date(2026, 5, 15, 12, 0, 0); // 15 June 2026

describe("parseBirthDate", () => {
  it("accepts an adult and computes age", () => {
    expect(parseBirthDate("15", "6", "1990", now)).toEqual({ ok: true, iso: "1990-06-15", age: 36 });
  });
  it("turns 18 exactly on the birthday", () => {
    expect(parseBirthDate("15", "6", "2008", now)).toMatchObject({ ok: true, age: 18 });
    expect(parseBirthDate("16", "6", "2008", now)).toMatchObject({ ok: false });
  });
  it("rejects minors with a clear message", () => {
    const r = parseBirthDate("1", "1", "2015", now);
    expect(r).toMatchObject({ ok: false });
    expect(r.ok === false && r.error).toMatch(/18/);
  });
  it("rejects impossible and malformed dates", () => {
    for (const [d, m, y] of [["31", "2", "1990"], ["0", "1", "1990"], ["1", "13", "1990"], ["a", "1", "1990"], ["1", "1", "90"], ["29", "2", "2023"]]) {
      expect(parseBirthDate(d, m, y, now).ok).toBe(false);
    }
    expect(parseBirthDate("29", "2", "2000", now).ok).toBe(true);
  });
  it("rejects absurd ages", () => {
    expect(parseBirthDate("1", "1", "1900", now).ok).toBe(false);
  });
});

describe("ageOn / splitIsoDate", () => {
  it("handles the day before a birthday", () => {
    expect(ageOn("2000-06-16", now)).toBe(25);
    expect(ageOn("2000-06-15", now)).toBe(26);
  });
  it("splits ISO dates for the edit form", () => {
    expect(splitIsoDate("1990-06-05")).toEqual({ day: "5", month: "6", year: "1990" });
  });
});

describe("formatListTime", () => {
  it("uses compact relative units", () => {
    expect(formatListTime(new Date(now.getTime() - 5_000).toISOString(), now)).toBe("now");
    expect(formatListTime(new Date(now.getTime() - 5 * 60_000).toISOString(), now)).toBe("5m");
    expect(formatListTime(new Date(now.getTime() - 3 * 3_600_000).toISOString(), now)).toBe("3h");
  });
});
