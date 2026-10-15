import { describe, expect, it } from "vitest";
import { ageFromIso, checkBirthDate, formatListTime, fromIsoDate, toIsoDate } from "./dates";

describe("toIsoDate", () => {
  it("accepts real dates and pads the output", () => {
    expect(toIsoDate({ day: "5", month: "3", year: "1994" })).toBe("1994-03-05");
    expect(toIsoDate({ day: "29", month: "2", year: "2000" })).toBe("2000-02-29");
  });

  it("rejects impossible or malformed dates", () => {
    expect(toIsoDate({ day: "31", month: "4", year: "1994" })).toBeNull();
    expect(toIsoDate({ day: "29", month: "2", year: "2001" })).toBeNull();
    expect(toIsoDate({ day: "0", month: "1", year: "1994" })).toBeNull();
    expect(toIsoDate({ day: "1", month: "13", year: "1994" })).toBeNull();
    expect(toIsoDate({ day: "1", month: "1", year: "94" })).toBeNull();
    expect(toIsoDate({ day: "a", month: "1", year: "1994" })).toBeNull();
    expect(toIsoDate({ day: "", month: "", year: "" })).toBeNull();
  });
});

describe("fromIsoDate", () => {
  it("round-trips with toIsoDate", () => {
    expect(fromIsoDate("1994-03-05")).toEqual({ day: "5", month: "3", year: "1994" });
    expect(toIsoDate(fromIsoDate("1994-03-05"))).toBe("1994-03-05");
  });
});

describe("age rules", () => {
  const now = new Date(2026, 9, 15); // 15 Oct 2026

  it("counts whole years", () => {
    expect(ageFromIso("2000-10-15", now)).toBe(26);
    expect(ageFromIso("2000-10-16", now)).toBe(25);
    expect(ageFromIso("2000-11-01", now)).toBe(25);
  });

  it("enforces 18+ and a sane upper bound", () => {
    expect(checkBirthDate({ day: "15", month: "10", year: "2008" }, now).status).toBe("ok");
    expect(checkBirthDate({ day: "16", month: "10", year: "2008" }, now).status).toBe("underage");
    expect(checkBirthDate({ day: "1", month: "1", year: "1900" }, now).status).toBe("too_old");
    expect(checkBirthDate({ day: "1", month: "1", year: "2030" }, now).status).toBe("invalid");
    expect(checkBirthDate({ day: "31", month: "2", year: "1990" }, now).status).toBe("invalid");
  });
});

describe("formatListTime", () => {
  const now = new Date(2026, 9, 15, 18, 0);

  it("shows the time for today, 'hier' for yesterday and a date for older messages", () => {
    expect(formatListTime(new Date(2026, 9, 15, 9, 5).toISOString(), now)).toMatch(/^\d{2}:\d{2}$/);
    expect(formatListTime(new Date(2026, 9, 14, 23, 0).toISOString(), now)).toBe("hier");
    expect(formatListTime(new Date(2026, 8, 1, 12, 0).toISOString(), now)).toMatch(/1/);
  });
});
